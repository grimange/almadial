package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/grimange/almadial/internal/artifacts"
)

const PhoneRevision = "24e98fccfa7b7079fff970e02fca5f002ac1f2cc"
const phoneArchiveSHA = "f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8"
const phoneIdentity = "alm_s10_9061"

var errPhone = errors.New("S10 phone contract failed; preserve existing identities and inspect the named postcondition")

// PhoneRequest is deliberately limited to the accepted, registration-only lab
// topology. It cannot authorize public SIP, media or campaign activity.
type PhoneRequest struct{ Identity, Hostname, WSSURL string }

func phoneRequestValid(r PhoneRequest) bool {
	return r == (PhoneRequest{phoneIdentity, "localhost", "wss://localhost:8089/ws"})
}

// Verified bytes are retained so activation cannot substitute another archive.
type PhoneSource struct{ files map[string][]byte }

func (s PhoneSource) Digest() string {
	if len(s.files) == 0 {
		return ""
	}
	return hash(s.files)
}
func LoadPhoneSource(ctx context.Context, m artifacts.Manifest, archive string) (PhoneSource, error) {
	spec, e := m.Artifact("viciphone")
	if e != nil {
		return PhoneSource{}, e
	}
	v, e := artifacts.VerifyLocal(ctx, spec, archive)
	if e != nil || v.Identity != artifacts.Verified {
		return PhoneSource{}, errPhone
	}
	b, e := sourceBytes(archive, phoneArchiveSHA)
	if e != nil {
		return PhoneSource{}, e
	}
	gz, e := gzip.NewReader(bytes.NewReader(b))
	if e != nil {
		return PhoneSource{}, errPhone
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return PhoneSource{}, errPhone
		}
		if e = ctx.Err(); e != nil {
			return PhoneSource{}, e
		}
		rel := strings.TrimPrefix(h.Name, "ViciPhone-"+PhoneRevision+"/")
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := ""
		if strings.HasPrefix(rel, "src/") {
			name = strings.TrimPrefix(rel, "src/")
		} else if rel == "LICENSE" {
			name = "LICENSE"
		}
		if name == "" {
			continue
		}
		data, e := io.ReadAll(io.LimitReader(tr, 4<<20))
		if e != nil || len(data) >= 4<<20 {
			return PhoneSource{}, errPhone
		}
		files[name] = data
	}
	for name, pin := range map[string]string{"viciphone.php": "e37d92df58d3c4f1b55a24a4364f9e4c9653c45820628fb657c42bd443fb6a6c", "js/vici_phone.js": "c7763fd3cc5a56d604c577291e6bbd25afe7f3b1cfa5fadbce914d30cf94779b", "js/sip.js": "9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b", "LICENSE": "e7ae8b5da17709474ba25316441bc064148841cf101e7eef318c3c3ee5a0827b"} {
		if rawSHA(files[name]) != pin {
			return PhoneSource{}, errPhone
		}
	}
	// Reuse the M0-qualified bounded compatibility changes: suppress sensitive
	// SIP diagnostics and let the visible Register control reconnect SIP.js.
	replacements := map[string][][2]string{
		"viciphone.php": {{"$debug_access_log = true;", "$debug_access_log = false;"}},
		"js/vici_phone.js": {{"logBuiltinEnabled: true", "logBuiltinEnabled: false"}, {"logConfiguration: true", "logConfiguration: false"}, {"logLevel: \"debug\"", "logLevel: \"error\""}, {"traceSip: true", "traceSip: false"},
			{"function registerButton( ) {\n\tdebugOut( 'Register Button Pressed' );\n\n\t// Send REGISTER\n\tmyRegisterer.register()", "function registerButton( ) {\n\tdebugOut( 'Register Button Pressed' );\n\n\t// SIP.js requires a connected transport before REGISTER.\n\tconst ready = myUserAgent.transport.state === SIP.TransportState.Disconnected\n\t\t? myUserAgent.reconnect() : Promise.resolve();\n\tready.then(() => myRegisterer.register({requestDelegate: {onAccept: handleRegRegistered}}))"},
			{"function handleRegRegistered() {", "function handleRegRegistered() {\n\tif (incall) uiElements.reg_status.value = lang.incall + ' - ' + caller;"}},
	}
	for name, changes := range replacements {
		text := string(files[name])
		for _, pair := range changes {
			if strings.Count(text, pair[0]) != 1 {
				return PhoneSource{}, errPhone
			}
			text = strings.Replace(text, pair[0], pair[1], 1)
		}
		files[name] = []byte(text)
	}
	return PhoneSource{files}, nil
}
