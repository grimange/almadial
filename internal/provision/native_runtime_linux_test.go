//go:build linux

package provision

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/journal"
)

func TestNativeCronAndSourceVerification(t *testing.T) {
	// Exercise the same embedded program used on the host, without host effects.
	program := strings.Replace(nativeRuntimeProgram(), "if __name__=='__main__': native_main()", "", 1)
	program += `
assert len(NR_UNITS)==4
assert str(NR_SECRET)=='/etc/astguiclient-native-ami.secret'
assert NR_SECRET.parent==Path('/etc')
assert b'User=asterisk' in native_unit('listener',NR_UNITS['listener'])
assert b'Restart=always' in native_unit('sender',NR_UNITS['sender'])
try:
    nr_source({'Sources':{'wrong':base64.b64encode(gzip.compress(b'wrong revision')).decode()},'SourcePins':{'wrong':'0'*64}},'wrong')
except ValueError:
    pass
else:
    raise AssertionError('wrong source revision accepted')
assert 'endpoint/webrtc=yes' in nr_phone_template({'Network':{'ManagementAddress':'192.0.2.3','RTPStart':10000,'RTPEnd':10100}})
print(native_cron().decode(),end='')
`
	out, err := exec.Command("python3", "-B", "-c", program).CombinedOutput()
	if err != nil {
		t.Fatalf("embedded program: %v: %s", err, out)
	}
	if string(out) != NativeRuntimeCron() {
		t.Fatal("diagnostics and installed cron differ")
	}
}

func TestNativeReadinessTransitionAndLabBoundary(t *testing.T) {
	for _, native := range []bool{false, true} {
		before := RuntimeSnapshot{Host: hash("host"), Foundation: hash("plan"), Foreign: hash("immutable"), Files: "absent", Services: "absent", Health: "absent"}
		after := before
		after.Files = hash("files")
		after.Services = hash("services")
		after.Health = hash("healthy")
		f := &runtimeFake{s: after}
		c := runtimeChecks{host: f, before: before, current: before, action: execution.Action{ID: "runtime.00.files"}}
		if native {
			c.policy = NativeRuntimePolicyDigest()
		}
		err := c.Check(context.Background(), "runtime.00.files.verified")
		if native && err != nil {
			t.Fatal("native missing-file reconciliation rejected", err)
		}
		if !native && err == nil {
			t.Fatal("lab transition contract weakened")
		}
		f.s.Foreign = hash("unexpected foreign mutation")
		if c.Check(context.Background(), "runtime.00.files.verified") == nil {
			t.Fatal("foreign mutation accepted")
		}
	}
}

func TestNativeFailedReadinessIsNeverSatisfiedNoOp(t *testing.T) {
	before := RuntimeSnapshot{Host: hash("host"), Foundation: hash("plan"), Foreign: hash("immutable"), Files: hash("files"), Services: hash("services"), Health: "absent"}
	if runtimeSatisfied(before) {
		t.Fatal("failed live health became NO_OP")
	}
	f := &runtimeFake{s: before}
	c := runtimeChecks{host: f, before: before, current: before, policy: NativeRuntimePolicyDigest()}
	observation, err := c.Observe(context.Background(), journal.Record{ID: "runtime.02.verify"})
	if err != nil || observation.State != journal.Absent {
		t.Fatal(observation, err)
	}
	f.s.Health = hash("healthy")
	observation, err = c.Observe(context.Background(), journal.Record{ID: "runtime.02.verify"})
	if err != nil || observation.State != journal.Satisfied || observation.Evidence == "" {
		t.Fatal(observation, err)
	}
}
