package operatorui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"

	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/plan"
	"github.com/grimange/almadial/internal/preflight"
)

var ErrOutput = errors.New("terminal output failed; check the output destination; no approval recorded")
var ErrCanceled = errors.New("preparation canceled; no approval recorded and no installation started")
var ErrInput = errors.New("terminal input ended or failed; rerun wizard in a terminal or use --unattended with a validated --config")
var ErrBlocked = errors.New("preparation blocked; resolve mandatory findings and manifest authorization, then review a fresh plan")
var ErrApproval = errors.New("approval denied or mismatched; explicitly approve the current configuration and plan")

// Prompts uses a single bounded line reader. Close the underlying terminal input
// on cancellation to interrupt an outstanding OS read. No input value is echoed.
// All prompts accept 'cancel'; EOF, partial final lines and Ctrl+C fail closed.
type Prompts struct {
	ctx    context.Context
	out    io.Writer
	reader *bufio.Reader
}
type line struct {
	text string
	err  error
}

func NewPrompts(ctx context.Context, in io.Reader, out io.Writer) *Prompts {
	return &Prompts{ctx: ctx, out: out, reader: bufio.NewReaderSize(in, 4096)}
}
func (p *Prompts) Ask(label, current string) (string, error) {
	if p.ctx.Err() != nil {
		return "", ErrCanceled
	}
	suffix := ""
	if current != "" {
		suffix = " [" + current + "]"
	}
	if _, e := fmt.Fprintf(p.out, "%s%s (cancel to stop): ", label, suffix); e != nil {
		return "", ErrOutput
	}
	lines := make(chan line, 1)
	go func() {
		s, e := p.reader.ReadSlice('\n')
		v := line{err: e}
		if e == nil {
			v.text = string(s)
		}
		lines <- v
	}()
	select {
	case <-p.ctx.Done():
		return "", ErrCanceled
	case v := <-lines:
		if p.ctx.Err() != nil {
			return "", ErrCanceled
		}
		if v.err != nil {
			return "", ErrInput
		}
		s := strings.TrimSpace(v.text)
		if strings.EqualFold(s, "cancel") || strings.ContainsRune(s, 3) {
			return "", ErrCanceled
		}
		if s == "" {
			s = current
		}
		return s, nil
	}
}
func (p *Prompts) YesNo(label string, defaultNo bool) (bool, error) {
	for {
		d := ""
		if defaultNo {
			d = "no"
		}
		s, e := p.Ask(label+" (yes/no)", d)
		if e != nil {
			return false, e
		}
		switch strings.ToLower(s) {
		case "yes", "y":
			return true, nil
		case "no", "n":
			return false, nil
		}
		if _, e = fmt.Fprintln(p.out, "Enter yes or no, or cancel."); e != nil {
			return false, ErrOutput
		}
	}
}

type field struct {
	key, label string
	get        func() string
	set        func(string) error
}

// Configure edits only the accepted typed config. Defaults and all semantic
// checks come from config; syntax errors never echo the submitted value.
func Configure(p *Prompts, c config.Config) (config.Config, error) {
	if c.SchemaVersion == 0 {
		c.SchemaVersion = config.SchemaVersion
	}
	c = config.Defaults(c)
	str := func(v *string) (func() string, func(string) error) {
		return func() string { return *v }, func(s string) error {
			if s == "-" {
				s = ""
			}
			*v = s
			return nil
		}
	}
	fields := []field{}
	addString := func(key, label string, v *string) { g, s := str(v); fields = append(fields, field{key, label, g, s}) }
	addIP := func(key, label string, v *netip.Addr) {
		fields = append(fields, field{key, label, func() string {
			if v.IsValid() {
				return v.String()
			}
			return ""
		}, func(s string) error {
			a, e := netip.ParseAddr(s)
			if e == nil {
				*v = a
			}
			return e
		}})
	}
	addPort := func(key, label string, v **int) {
		fields = append(fields, field{key, label, func() string {
			if *v == nil {
				return ""
			}
			return strconv.Itoa(**v)
		}, func(s string) error {
			n, e := strconv.Atoi(s)
			if e == nil {
				*v = &n
			}
			return e
		}})
	}
	addIP("network.bind_address", "Target server bind IP", &c.Network.BindAddress)
	addIP("network.advertise_address", "Advertised IP (required for wildcard bind)", &c.Network.AdvertiseAddress)
	addString("network.hostname", "Optional hostname (- clears; IP-only is supported)", &c.Network.Hostname)
	addPort("sip.port", "SIP UDP port", &c.SIP.Port)
	addPort("rtp.start", "RTP first UDP port", &c.RTP.Start)
	addPort("rtp.end", "RTP last UDP port", &c.RTP.End)
	fields = append(fields, field{"webrtc.enabled", "Enable optional WebRTC (yes/no)", func() string {
		if c.WebRTC.Enabled {
			return "yes"
		}
		return "no"
	}, func(s string) error {
		switch strings.ToLower(s) {
		case "yes", "y":
			c.WebRTC.Enabled = true
		case "no", "n":
			c.WebRTC.Enabled = false
			c.WebRTC.WSSPort = nil
			c.WebRTC.TLS = nil
		default:
			return errors.New("enter yes or no")
		}
		return nil
	}})
	for _, r := range []struct {
		key, label string
		v          *config.SecretRef
	}{{"database", "Database", &c.Credentials.Database}, {"ami", "AMI", &c.Credentials.AMI}, {"sip", "SIP", &c.Credentials.SIP}, {"admin", "Administrator", &c.Credentials.Admin}} {
		ref := r.v
		fields = append(fields, field{"credentials." + r.key, r.label + " credential reference (env:NAME or file:/absolute/path; NEVER a password)", func() string {
			if ref.Source != "" {
				return string(ref.Type) + ":" + ref.Source
			}
			return ""
		}, func(s string) error {
			kind, source, ok := strings.Cut(s, ":")
			if !ok {
				return errors.New("reference required")
			}
			*ref = config.SecretRef{Type: config.ReferenceType(kind), Source: source}
			return nil
		}})
	}
	ask := func(f field) error {
		for {
			s, e := p.Ask(f.label, f.get())
			if e != nil {
				return e
			}
			if e = f.set(s); e == nil {
				return nil
			}
			if _, e = fmt.Fprintln(p.out, "Invalid input for "+f.key+"; correct the requested value."); e != nil {
				return ErrOutput
			}
		}
	}
	if _, e := fmt.Fprintln(p.out, "AlmaDial configuration: Enter keeps the displayed value; '-' clears optional text.\nCredential inputs are references only. Values are never resolved or saved here."); e != nil {
		return c, ErrOutput
	}
	// Normalize after each answer so advertised IP follows an explicit bind address
	// and optional WSS defaults remain owned by config.Defaults.
	for i := range fields {
		if e := ask(fields[i]); e != nil {
			return c, e
		}
		c = config.Defaults(c)
	}
	if c.WebRTC.Enabled {
		c = config.Defaults(c)
		if c.WebRTC.TLS == nil {
			c.WebRTC.TLS = &config.TLS{}
		}
		addPort("webrtc.wss_port", "WSS TCP port", &c.WebRTC.WSSPort)
		addString("webrtc.tls.certificate_file", "TLS certificate file reference", &c.WebRTC.TLS.CertificateFile)
		addString("webrtc.tls.private_key_file", "TLS private-key file reference (NEVER key contents)", &c.WebRTC.TLS.PrivateKeyFile)
		for _, f := range fields[len(fields)-3:] {
			if e := ask(f); e != nil {
				return c, e
			}
		}
	}
	for {
		effective := config.Defaults(c)
		e := config.Validate(effective)
		if e == nil {
			return effective, nil
		}
		if _, werr := fmt.Fprintln(p.out, "Configuration invalid: "+e.Error()); werr != nil {
			return c, ErrOutput
		}
		var fe *config.FieldError
		if !errors.As(e, &fe) {
			return c, e
		}
		found := false
		for _, f := range fields {
			if fe.Field == f.key || strings.HasPrefix(fe.Field, f.key+".") {
				found = true
				if err := ask(f); err != nil {
					return c, err
				}
				break
			}
		}
		if !found {
			return c, e
		}
	}
}

// Preparation contains the accepted immutable plan, never an alternative plan.
type Preparation struct {
	Config   config.Config
	Plan     plan.Plan
	Manifest artifacts.Manifest
}

func Prepare(c config.Config, s preflight.Snapshot, r preflight.Requirements, m artifacts.Manifest) (Preparation, error) {
	c = config.Defaults(c)
	if e := config.Validate(c); e != nil {
		return Preparation{}, e
	}
	p, e := plan.New(c, s, r, m.Digest())
	return Preparation{c, p, m}, e
}
func (p Preparation) Present(w io.Writer) error {
	human, e := p.Plan.Human()
	if e != nil {
		return e
	}
	_, e = fmt.Fprintf(w, "AlmaDial - Installation Preparation\nConfiguration: VALID\nTarget IP: %s\nOptional hostname: %s\nWebRTC enabled: %t\nManifest status: %s\nArtifact verification: NOT_RUN (manifest identity checked only)\nInstallation authorization: %t\nExecution: NOT STARTED\n%s", p.Config.Network.AdvertiseAddress, p.Config.Network.Hostname, p.Config.WebRTC.Enabled, strings.ToUpper(p.Manifest.Status()), p.Manifest.InstallationAuthorized(), human)
	if e != nil {
		return e
	}
	for _, f := range p.Plan.Findings() {
		if f.Blocking {
			if _, e = fmt.Fprintf(w, "Action for %s: investigate the finding above; installation cannot continue. Rerun preparation after correction.\n", f.ID); e != nil {
				return ErrOutput
			}
		}
	}
	return nil
}

// Decision is transient review context, not a durable execution capability.
// Future execution must revalidate all inputs/authorization and obtain consent
// at its mutation boundary. Digest equality identifies input, never a person.
type Decision struct {
	Approved                 bool
	ConfigSHA256, PlanSHA256 string
}

func (d Decision) Validate(p Preparation) error {
	return validateDecision(p, d, p.Manifest.InstallationAuthorized())
}
func validateDecision(p Preparation, d Decision, authorized bool) error {
	if e := config.ValidateUnattended(p.Config, config.Approval{Approved: d.Approved, ConfigSHA256: d.ConfigSHA256}); e != nil {
		return e
	}
	// Bind config to the actual immutable plan, including host and manifest identity.
	cd, e := config.Digest(p.Config)
	if e != nil {
		return e
	}
	if p.Plan.ConfigSHA256() != cd || p.Plan.ManifestSHA256() != p.Manifest.Digest() {
		return ErrApproval
	}
	if !p.Plan.Eligible() || !authorized {
		return ErrBlocked
	}
	if d.PlanSHA256 == "" || d.PlanSHA256 != p.Plan.Digest() {
		return ErrApproval
	}
	return nil
}
func (p Preparation) Approve(ctx context.Context, prompts *Prompts) (Decision, error) {
	return p.approve(ctx, prompts, p.Manifest.InstallationAuthorized())
}
func (p Preparation) approve(ctx context.Context, prompts *Prompts, authorized bool) (Decision, error) {
	if !p.Plan.Eligible() || !authorized {
		return Decision{}, ErrBlocked
	}
	cd, e := config.Digest(p.Config)
	if e != nil {
		return Decision{}, e
	}
	if _, e = fmt.Fprintf(prompts.out, "Review context: config %s\nPlan %s\nApproval concerns future mutating operations and their listed risks.\nThis command records a transient decision only; no installation will run.\n", cd, p.Plan.Digest()); e != nil {
		return Decision{}, ErrOutput
	}
	yes, e := prompts.YesNo("Approve this exact plan for a future mutating operation?", false)
	if e != nil {
		return Decision{}, e
	}
	if ctx.Err() != nil {
		return Decision{}, ErrCanceled
	}
	d := Decision{yes, cd, p.Plan.Digest()}
	if e = validateDecision(p, d, authorized); e != nil {
		return Decision{}, e
	}
	return d, nil
}
