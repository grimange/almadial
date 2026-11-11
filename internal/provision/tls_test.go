package provision

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func tlsFixture(t *testing.T, host string, expiry time.Time) ([]byte, []byte, []byte) {
	t.Helper()
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	caKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "S08 test CA"}, NotBefore: now.Add(-24 * time.Hour), NotAfter: now.Add(365 * 24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, e := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if e != nil {
		t.Fatal(e)
	}
	trust := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), DNSNames: []string{host}, NotBefore: now.Add(-time.Hour), NotAfter: expiry, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, e = x509.CreateCertificate(rand.Reader, leaf, ca, &key.PublicKey, caKey)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := x509.MarshalPKCS8PrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}), trust
}
func TestTLSCertificateValidation(t *testing.T) {
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	chain, key, trust := tlsFixture(t, "localhost", now.Add(time.Hour))
	_, otherKey, otherCA := tlsFixture(t, "localhost", now.Add(time.Hour))
	expired, expiredKey, expiredCA := tlsFixture(t, "localhost", now.Add(-time.Minute))
	for _, tc := range []struct {
		name, host        string
		chain, key, trust []byte
		now               time.Time
		valid             bool
	}{
		{"trusted private CA", "localhost", chain, key, trust, now, true},
		{"wrong hostname", "other.invalid", chain, key, trust, now, false},
		{"wrong key", "localhost", chain, otherKey, trust, now, false},
		{"untrusted chain", "localhost", chain, key, otherCA, now, false},
		{"expired", "localhost", expired, expiredKey, expiredCA, now, false},
		{"not yet valid", "localhost", chain, key, trust, now.Add(-2 * time.Hour), false},
		{"invalid certificate", "localhost", []byte("invalid"), key, trust, now, false},
		{"invalid anchors", "localhost", chain, key, []byte("invalid"), now, false},
		{"key in public chain", "localhost", append(append([]byte{}, chain...), key...), key, trust, now, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, e := ValidateTLSMaterial(tc.host, tc.chain, tc.key, tc.trust, tc.now)
			if (e == nil) != tc.valid {
				t.Fatal(c, e)
			}
			if e != nil && strings.Contains(e.Error(), "PRIVATE KEY") {
				t.Fatal("key leaked")
			}
		})
	}
	c, e := ValidateTLSMaterial("localhost", chain, key, trust, now)
	if e != nil {
		t.Fatal(e)
	}
	again, _ := ValidateTLSMaterial("localhost", chain, key, trust, now.Add(time.Minute))
	if c != again {
		t.Fatal("nondeterministic certificate identity")
	}
}
