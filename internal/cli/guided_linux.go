//go:build linux

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/operatorui"
	"github.com/grimange/almadial/internal/preflight"
	"github.com/grimange/almadial/internal/provision"
	"github.com/grimange/almadial/internal/releaseassets"
)

const guidedRoot = "/var/lib/almadial-inputs"

func inputDigest(s string) string { b := sha256.Sum256([]byte(s)); return hex.EncodeToString(b[:]) }

// Protected input storage is separate from the existing execution journal. It
// refuses foreign directories, symlinks and overwrites, including on retries.
func guidedDirectory(path string) error {
	if err := os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	f, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := os.Lstat(path)
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		return errors.New("guided inputs require an owned private directory (0700)")
	}
	opened, err := f.Stat(".")
	if err != nil || !os.SameFile(st, opened) {
		return errors.New("guided input directory changed")
	}
	// Reuse the artifact cache's strict current-owner check before writing inputs.
	return guidedOwned(st)
}
func guidedSave(path string, b []byte) error {
	if old, err := guidedRead(path); err == nil {
		if bytes.Equal(old, b) {
			return nil
		}
		return errors.New("saved installation input differs; preserve it for reconciliation")
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, e := f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	return e
}

func guidedRead(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() > 32<<20 {
		return nil, errors.New("saved input must be a private regular file (0600)")
	}
	if err = guidedOwned(st); err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(f, 32<<20+1))
}

// approvedInstall passes the displayed plan identity directly to the existing
// executor; no second command or manual digest copy is involved.
func approveInstall(p *operatorui.Prompts, out io.Writer, digest string) error {
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size {
		return errors.New("invalid installation plan identity")
	}
	if _, err := fmt.Fprintln(out, "Installation plan SHA-256:", digest); err != nil {
		return err
	}
	yes, err := p.YesNo("Approve this exact plan and execute the installation", true)
	if err != nil {
		return err
	}
	if !yes {
		return operatorui.ErrApproval
	}
	return nil
}

// A fresh installation uses the reviewed release defaults. Mirrors are advanced
// overrides, not mandatory operator inputs. The caller preserves an existing
// saved policy before considering this function, so retries never change origins.
func guidedOrigins(m artifacts.Manifest, store string) ([]byte, error) {
	if store != "" {
		return artifacts.AcquisitionPolicy(m, store)
	}
	if _, err := artifacts.WithAcquisitionOrigins(m, bytes.NewReader(releaseassets.Origins)); err != nil {
		return nil, err
	}
	return append([]byte(nil), releaseassets.Origins...), nil
}

func runGuidedInstall(ctx context.Context, in io.Reader, out, errout io.Writer) int {
	fail := func(err error) int {
		fmt.Fprintln(errout, "almadial installation:", err, "\nCompleted stage state is preserved. Correct the reported prerequisite and rerun almadial install; do not delete the journal.")
		return Failure
	}
	if !terminal(in) || !terminal(out) {
		fmt.Fprintln(errout, "almadial: installation inputs required without an interactive terminal; automation uses install --plan --manifest --approve-sha256")
		return Usage
	}
	if os.Geteuid() != 0 {
		return fail(errors.New("guided installation requires local root on the authorized target"))
	}
	fmt.Fprintln(out, "AlmaDial: single-server AlmaLinux 10.2 installation. Recommended qualified baseline: 4 CPUs, 8 GiB RAM, 80 GiB disk. Enforced fresh-install thresholds: 4 CPUs, 7 GiB total RAM, 4 GiB available RAM, 60 GiB free on installation filesystems, plus 1 GiB recording reserve and 10,000 recording inodes. Choose remote HTTPS access or the isolated laboratory profile. Carrier configuration is outside this installer.")
	prompts := operatorui.NewPrompts(ctx, in, out)
	if err := guidedDirectory(guidedRoot); err != nil {
		return fail(err)
	}
	manifestPath := filepath.Join(guidedRoot, "stack.lock.yaml")
	if err := guidedSave(manifestPath, releaseassets.Manifest); err != nil {
		return fail(err)
	}
	m, err := artifacts.Load(bytes.NewReader(releaseassets.Manifest))
	if err != nil {
		return fail(err)
	}
	originPath := filepath.Join(guidedRoot, "acquisition-origins.json")
	var originBytes []byte
	if saved, e := guidedRead(originPath); e == nil {
		originBytes = saved
	} else if !os.IsNotExist(e) {
		return fail(e)
	} else {
		originBytes, e = guidedOrigins(m, os.Getenv("ALMADIAL_ARTIFACT_STORE"))
		if e != nil {
			return fail(e)
		}
		if e = guidedSave(originPath, originBytes); e != nil {
			return fail(e)
		}
	}
	m, err = artifacts.WithAcquisitionOrigins(m, bytes.NewReader(originBytes))
	if err != nil {
		return fail(err)
	}
	planPath := filepath.Join(guidedRoot, "plan.json")
	if raw, e := guidedRead(planPath); e == nil {
		request, e := provision.InstallPlanRequest(raw)
		if e != nil {
			return fail(e)
		}
		if e = guidedDirectory(filepath.Join(guidedRoot, "artifacts")); e != nil {
			return fail(e)
		}
		paths, e := artifacts.AcquireInstall(ctx, m, filepath.Join(guidedRoot, "artifacts"), out)
		if e != nil {
			return fail(e)
		}
		// Reacquisition must recover the same plan-bound paths and exact content.
		if request.Sources.VICIdial != paths["vicidial"] || request.Sources.Asterisk != paths["asterisk"] || request.Sources.Perl != paths["net-telnet"] || request.Sources.PJProject != paths["pjproject"] || request.Sources.Phone != paths["viciphone"] {
			return fail(errors.New("saved source paths differ; preserve original plan for reconciliation"))
		}
		sources, e := provision.LoadInstallSources(ctx, m, request.Sources)
		if e != nil {
			return fail(e)
		}
		plan, e := provision.DecodeInstallPlan(bytes.TrimSpace(raw), m, sources)
		if e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, "Saved installation plan: revalidate completed stages and resume remaining work; a completed host must verify NO_OP.")
		fmt.Fprintf(out, "Profile: %s; HTTPS hostname %s; management %s on %s; install pinned database, telephony, web and browser-phone components.\n", request.Network.Mode, request.TLS.Hostname, request.Network.ManagementAddress, request.Network.Interface)
		if request.Authorization.Mode == provision.StandardAuthorization {
			if e = provision.VerifyManagement(ctx, request.Network, request.TLS.Hostname); e != nil {
				return fail(e)
			}
		}
		if e = reviewInstall(prompts, out, plan); e != nil {
			return fail(e)
		}
		result := runInstall(ctx, []string{"resume", "--plan", planPath, "--manifest", manifestPath, "--approve-sha256", plan.Digest()}, out, errout)
		if result == Success {
			guidedCompletion(out, request.Network.Mode, request.TLS.Hostname)
		}
		return result
	} else if !os.IsNotExist(e) {
		return fail(e)
	}
	profile, err := prompts.Ask("Deployment profile: remote (direct HTTPS/WSS and UDP media) or lab (loopback)", "remote")
	if err != nil {
		return fail(err)
	}
	if profile != "remote" && profile != "lab" {
		return fail(errors.New("choose remote or lab"))
	}
	hostname := "localhost"
	if profile == "remote" {
		hostname, err = prompts.Ask("Server FQDN matching the supplied trusted certificate", "")
		if err != nil {
			return fail(err)
		}
	}
	n, err := provision.NewNative(nil)
	if err != nil {
		return fail(err)
	}
	inventory, err := n.Inspect(ctx)
	if err != nil {
		return fail(err)
	}
	snapshotHost := preflight.Collect(ctx, preflight.Linux{})
	pairs, err := provision.DiscoverIPv4Interfaces(ctx)
	if err != nil {
		return fail(err)
	}
	defaultIP := ""
	if len(pairs) == 1 {
		defaultIP = pairs[0].Address
	} else {
		fmt.Fprintln(out, "Assigned IPv4/interface choices (configured addresses do not prove public reachability):")
		for _, pair := range pairs {
			fmt.Fprintf(out, "  %s on %s\n", pair.Address, pair.Interface)
		}
	}
	address, err := prompts.Ask("Directly attached server IPv4 management/media address", defaultIP)
	if err != nil {
		return fail(err)
	}
	defaultInterface := ""
	for _, pair := range pairs {
		if pair.Address == address {
			if defaultInterface != "" {
				defaultInterface = ""
				break
			}
			defaultInterface = pair.Interface
		}
	}
	iface, err := prompts.Ask("Management network interface for selected IPv4", defaultInterface)
	if err != nil {
		return fail(err)
	}
	sshPort, err := provision.DiscoverManagementSSH(ctx, address)
	if err != nil {
		return fail(err)
	}
	fmt.Fprintf(out, "Preserve existing SSH management port: %d/tcp\n", sshPort)
	tlsDefault := "lab"
	if profile == "remote" {
		tlsDefault = "provided"
	}
	tlsMode, err := prompts.Ask("TLS: provided (certificate/key/CA files), or lab (localhost private CA only)", tlsDefault)
	if err != nil {
		return fail(err)
	}
	tlsRequest := provision.TLSRequest{Mode: "provided", Hostname: hostname}
	if tlsMode == "lab" && profile == "lab" {
		tlsRequest, err = guidedLabTLS(guidedRoot)
	} else if tlsMode == "provided" {
		tlsRequest.CertificateFile, err = prompts.Ask("Hostname certificate full-chain PEM absolute path", "")
		if err == nil {
			tlsRequest.PrivateKeyFile, err = prompts.Ask("Matching private key PEM absolute path", "")
		}
		if err == nil {
			tlsRequest.TrustFile, err = prompts.Ask("Issuing CA trust PEM absolute path", "")
		}
	} else {
		err = errors.New("choose lab or provided TLS")
	}
	if err != nil {
		return fail(err)
	}
	if profile == "remote" {
		tlsRequest.Mode = "remote"
	}
	if _, err = (provision.TLSNative{Request: tlsRequest}).Material(); err != nil {
		return fail(err)
	}
	ip, err := netip.ParseAddr(address)
	if err != nil || !ip.Is4() {
		return fail(errors.New("select a valid assigned IPv4 address"))
	}
	c := config.Config{SchemaVersion: 1, Network: config.Network{BindAddress: ip, AdvertiseAddress: ip, Hostname: hostname}, WebRTC: config.WebRTC{Enabled: true, TLS: &config.TLS{CertificateFile: tlsRequest.CertificateFile, PrivateKeyFile: tlsRequest.PrivateKeyFile}}}
	c.Credentials = config.Credentials{Database: config.SecretRef{Type: config.File, Source: "/etc/almadial/secrets/database"}, AMI: config.SecretRef{Type: config.File, Source: "/etc/almadial/secrets/vicidial-ami"}, SIP: config.SecretRef{Type: config.File, Source: "/etc/almadial/secrets/phone-s10"}, Admin: config.SecretRef{Type: config.File, Source: "/etc/almadial/secrets/admin"}}
	c = config.Defaults(c)
	network := provision.NetworkRequest{Mode: "isolated", Interface: iface, ManagementAddress: address, SSHPort: sshPort}
	phone := provision.PhoneRequest{Identity: "alm_s10_9061", Hostname: "localhost", WSSURL: "wss://localhost:8089/ws"}
	if profile == "remote" {
		first, e := prompts.Ask("RTP UDP first port (even, at least 1024)", "10000")
		if e != nil {
			return fail(e)
		}
		last, e := prompts.Ask("RTP UDP last port (open this exact range on the server network)", "10019")
		if e != nil {
			return fail(e)
		}
		start, e1 := strconv.ParseUint(first, 10, 16)
		end, e2 := strconv.ParseUint(last, 10, 16)
		if e1 != nil || e2 != nil || start < 1024 || start%2 != 0 || end <= start {
			return fail(errors.New("invalid RTP UDP range"))
		}
		startInt := int(start)
		c.RTP.Start = &startInt
		endInt := int(end)
		c.RTP.End = &endInt
		network.Mode = "remote"
		network.RTPStart = uint16(start)
		network.RTPEnd = uint16(end)
		phone.Hostname = hostname
		phone.WSSURL = "wss://" + hostname + "/ws"
	}

	if err = provision.VerifyManagement(ctx, network, hostname); err != nil {
		return fail(err)
	}

	fmt.Fprintln(out, "Credentials will be securely generated by the existing provisioning stages and retained in root-only files under /etc/almadial/secrets. They are never printed in the installation log.")
	r := provision.InstallRequest{Config: c, Requirements: preflight.Requirements{Profile: "ALM46-single-server-not-production-sizing", MinCPUs: 4, CPUFlags: []string{"sse2"}, MinTotalMemory: 7516192768, MinAvailableMemory: 4294967296, MinDiskAvailable: 64424509440}, TLS: tlsRequest, Network: network, Phone: phone, Authorization: provision.InstallAuthorization{Mode: provision.StandardAuthorization, Host: inventory.Host}}
	// Read-only mount discovery; no storage changes or shell interpolation.
	mountCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	b, err := exec.CommandContext(mountCtx, "/usr/bin/findmnt", "--json", "--target", "/var", "--output", "TARGET,SOURCE,FSTYPE").Output()
	if err != nil {
		return fail(errors.New("cannot discover recording filesystem"))
	}
	var mounts struct {
		Filesystems []struct{ Target, Source, Fstype string }
	}
	if json.Unmarshal(b, &mounts) != nil || len(mounts.Filesystems) != 1 {
		return fail(errors.New("cannot identify recording filesystem"))
	}
	mount := mounts.Filesystems[0]
	r.Recording = provision.RecordingBudget{Profile: "ALM46-recording-reserve-not-production-sizing", Mount: mount.Target, Source: mount.Source, Filesystem: mount.Fstype, MinAvailableBytes: 1073741824, MinFreeInodes: 10000}
	cache := filepath.Join(guidedRoot, "artifacts")
	if err = guidedDirectory(cache); err != nil {
		return fail(err)
	}
	paths, err := artifacts.AcquireInstall(ctx, m, cache, out)
	if err != nil {
		return fail(err)
	}
	scratch := filepath.Join(guidedRoot, "scratch")
	if err = guidedDirectory(scratch); err != nil {
		return fail(err)
	}
	keyring := filepath.Join(guidedRoot, "asterisk-keyring.gpg")
	sig := filepath.Join(guidedRoot, "asterisk.asc")
	if err = guidedSave(keyring, releaseassets.Keyring); err != nil {
		return fail(err)
	}
	if err = guidedSave(sig, releaseassets.Signature); err != nil {
		return fail(err)
	}
	r.Sources = provision.InstallSourcePaths{VICIdial: paths["vicidial"], Asterisk: paths["asterisk"], Perl: paths["net-telnet"], PJProject: paths["pjproject"], Phone: paths["viciphone"], Signature: sig, Keyring: keyring, Scratch: scratch}
	sources, err := provision.LoadInstallSources(ctx, m, r.Sources)
	if err != nil {
		return fail(err)
	}
	cert, err := (provision.TLSNative{Request: r.TLS}).Material()
	if err != nil {
		return fail(err)
	}
	p, err := provision.PrepareInstall(r, provision.Inputs{Config: c, Requirements: r.Requirements, Preflight: snapshotHost, Inventory: inventory}, m, sources, cert)
	if err != nil {
		return fail(err)
	}
	fmt.Fprintln(out, "Plan: install pinned signed RPMs, local database/schema, Asterisk, VICIdial, Apache/PHP, profile-bound TLS, WSS and firewall; generate protected credentials and configure synthetic phone; retain durable journals and verify actual postconditions. The discovered SSH management port remains available. Browser/media and reboot acceptance require the separate operator run.")
	if err = reviewInstall(prompts, out, p); err != nil {
		return fail(err)
	}
	if err = guidedSave(planPath, p.JSON()); err != nil {
		return fail(err)
	}
	result := runInstall(ctx, []string{"install", "--plan", planPath, "--manifest", manifestPath, "--approve-sha256", p.Digest()}, out, errout)
	if result == Success {
		guidedCompletion(out, network.Mode, hostname)
	}
	return result
}

func reviewInstall(p *operatorui.Prompts, out io.Writer, plan provision.InstallPlan) error {
	if _, err := fmt.Fprintln(out, "AlmaDial will install or modify system packages, services, database configuration, certificates and firewall rules. Supported interrupted installations can be resumed, but AlmaDial does not guarantee complete system rollback. Maintain an appropriate backup or recovery method before proceeding."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, string(plan.JSON())); err != nil {
		return err
	}
	return approveInstall(p, out, plan.Digest())
}

func guidedCompletion(out io.Writer, profile, hostname string) {
	fmt.Fprintf(out, "Installed profile: %s; hostname: %s\nAccess: https://%s/vicidial/admin.php ; https://%s/agc/vicidial.php ; https://%s/almadial-phone/\n", profile, hostname, hostname, hostname, hostname)
	if profile == "isolated" {
		fmt.Fprintln(out, "Lab HTTPS is loopback only; use the documented SSH tunnel and trust the application CA.")
	}
	fmt.Fprintln(out, "Status and Doctor report actual health; warnings require review and required failures require correction:")
	for _, command := range []string{"status", "doctor"} {
		fmt.Fprintf(out, "almadial %s --plan /var/lib/almadial-inputs/plan.json --manifest /var/lib/almadial-inputs/stack.lock.yaml --ca-file /etc/pki/tls/almadial-s08/current/trust.pem\n", command)
	}
	fmt.Fprintln(out, "Resume: almadial install (or almadial resume). Review the same saved plan; completed stages are reconciled. Preserve inputs and journals. Changed TLS material, foreign files or unsafe interrupted database operations require operator reconciliation; no complete rollback is guaranteed.")
}
