//go:build linux

package cli

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/grimange/almadial/internal/provision"
)

func guidedOwned(st os.FileInfo) error {
	s, ok := st.Sys().(*syscall.Stat_t)
	if !ok || s.Uid != uint32(os.Geteuid()) {
		return errors.New("guided input must belong to current user")
	}
	return nil
}

// Private CA issuance is an explicit lab-only operator choice. The CA key is not
// retained; public trust must be imported into the operator browser. No trust
// store changes, verification bypass or public ACME dependency are introduced.
func guidedLabTLS(root string) (provision.TLSRequest, error) {
	dir := filepath.Join(root, "tls")
	r := provision.TLSRequest{Mode: "provided", Hostname: "localhost", CertificateFile: filepath.Join(dir, "cert.pem"), PrivateKeyFile: filepath.Join(dir, "key.pem"), TrustFile: filepath.Join(dir, "trust.pem")}
	if err := guidedDirectory(dir); err != nil {
		return r, err
	}
	if _, err := os.Lstat(r.CertificateFile); err == nil {
		_, err = guidedTLSIdentity(r)
		return r, err
	} else if !os.IsNotExist(err) {
		return r, err
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return r, err
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return r, err
	}
	serial := func() (*big.Int, error) { return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128)) }
	caSerial, err := serial()
	if err != nil {
		return r, err
	}
	leafSerial, err := serial()
	if err != nil {
		return r, err
	}
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: caSerial, Subject: pkix.Name{CommonName: "AlmaDial private laboratory CA"}, NotBefore: now.Add(-5 * time.Minute), NotAfter: now.Add(90 * 24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return r, err
	}
	leaf := &x509.Certificate{SerialNumber: leafSerial, Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: now.Add(-5 * time.Minute), NotAfter: now.Add(30 * 24 * time.Hour), BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		return r, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		return r, err
	}
	for _, f := range []struct {
		path, kind string
		data       []byte
	}{{r.PrivateKeyFile, "PRIVATE KEY", keyDER}, {r.TrustFile, "CERTIFICATE", caDER}, {r.CertificateFile, "CERTIFICATE", leafDER}} {
		if err = guidedSave(f.path, pem.EncodeToMemory(&pem.Block{Type: f.kind, Bytes: f.data})); err != nil {
			return r, err
		}
	}
	_, err = guidedTLSIdentity(r)
	return r, err
}

func guidedTLSIdentity(r provision.TLSRequest) (provision.TLSCertificate, error) {
	chain, err := os.ReadFile(r.CertificateFile)
	if err != nil {
		return provision.TLSCertificate{}, err
	}
	key, err := os.ReadFile(r.PrivateKeyFile)
	if err != nil {
		return provision.TLSCertificate{}, err
	}
	trust, err := os.ReadFile(r.TrustFile)
	if err != nil {
		return provision.TLSCertificate{}, err
	}
	return provision.ValidateTLSMaterial(r.Hostname, chain, key, trust, time.Now())
}
