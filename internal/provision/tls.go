package provision

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"strings"
	"time"
)

var errTLS = errors.New("S08 TLS contract failed; preserve the working generation and inspect the named TLS postcondition")

// Supplied certificates are the M0-qualified mode. Issuance and external ACME
// challenge authority are deliberately separate from local deployment.
type TLSRequest struct {
	Mode, Hostname, CertificateFile, PrivateKeyFile, TrustFile string
}

// TLSCertificate contains public identities only, never private key material.
type TLSCertificate struct {
	LeafSHA256, ChainSHA256, TrustSHA256 string
	NotBefore, NotAfter                  time.Time
}

func (c TLSCertificate) Digest() string { return hash(c) }

func ValidateTLSMaterial(host string, chain, key, trust []byte, now time.Time) (TLSCertificate, error) {
	fail := func(reason string) (TLSCertificate, error) {
		return TLSCertificate{}, errors.New("S08 certificate: " + reason)
	}
	if host == "" || strings.TrimSpace(host) != host || strings.ContainsAny(host, "/\\\n\r\t \"'") {
		return fail("invalid endpoint identity")
	}
	pair, e := tls.X509KeyPair(chain, key)
	if e != nil || len(pair.Certificate) == 0 {
		return fail("invalid certificate or mismatched private key")
	}
	leaf, e := x509.ParseCertificate(pair.Certificate[0])
	if e != nil {
		return fail("invalid leaf certificate")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(trust) {
		return fail("invalid trust anchors")
	}
	intermediates := x509.NewCertPool()
	for _, der := range pair.Certificate[1:] {
		cert, e := x509.ParseCertificate(der)
		if e != nil {
			return fail("invalid intermediate")
		}
		intermediates.AddCert(cert)
	}
	if _, e = leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: roots, Intermediates: intermediates, CurrentTime: now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}); e != nil {
		return fail("trust, SAN, validity or server-auth verification failed")
	}
	// Reject keys hidden in public chain/trust files and trailing non-PEM data.
	for _, input := range [][]byte{chain, trust} {
		for len(strings.TrimSpace(string(input))) > 0 {
			block, rest := pem.Decode(input)
			if block == nil || block.Type != "CERTIFICATE" {
				return fail("public material must contain certificates only")
			}
			input = rest
		}
	}
	digest := sha256.Sum256(leaf.Raw)
	return TLSCertificate{hex.EncodeToString(digest[:]), rawSHA(chain), rawSHA(trust), leaf.NotBefore.UTC(), leaf.NotAfter.UTC()}, nil
}

const tlsRoot = "/etc/pki/tls/almadial-s08"
const tlsLoaderPath = "/etc/httpd/conf.modules.d/99-almadial-s08.conf"
const tlsVhostPath = "/etc/httpd/almadial-s08.conf"
const tlsLoader = "# AlmaDial S08 TLS loader\nInclude /etc/httpd/almadial-s08.conf\n"
const tlsAsteriskBase = "#include almadial-s04-http.conf\n"
const tlsAsteriskHTTP = tlsAsteriskBase + "[general](+)\ntlsenable=yes\ntlsbindaddr=127.0.0.1:8089\ntlscertfile=" + tlsRoot + "/current/fullchain.pem\ntlsprivatekey=" + tlsRoot + "/current/key.pem\n"

func TLSVhost(host string) string {
	return "# AlmaDial S08: trusted local HTTPS\nListen 127.0.0.1:443 https\n" +
		strings.Replace(strings.Replace(WebVhost(), "<VirtualHost 127.0.0.1:80>", "<VirtualHost 127.0.0.1:443>\n    SSLEngine on\n    SSLProtocol -all +TLSv1.2 +TLSv1.3\n    SSLCertificateFile "+tlsRoot+"/current/fullchain.pem\n    SSLCertificateKeyFile "+tlsRoot+"/current/key.pem", 1), "ServerName localhost", "ServerName "+host, 1)
}
