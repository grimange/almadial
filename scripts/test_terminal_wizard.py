#!/usr/bin/env python3
"""Exercise the compiled M1-07 wizard using real Linux PTYs, synthetic config only.

Usage: python3 scripts/test_terminal_wizard.py --binary /tmp/almadial --output /tmp/pty
No VM, root, installation, network mutation or carrier access is used.
"""
import argparse
import errno
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import signal
import struct
import subprocess
import tempfile
import termios
import time

ROOT = Path(__file__).resolve().parent.parent


def run_pty(binary, args, steps, width=80, redirect=False):
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 24, width, 0, 0))

    def attach():
        os.setsid()
        fcntl.ioctl(slave, termios.TIOCSCTTY, 0)

    proc = subprocess.Popen([str(binary), *args], stdin=slave,
                            stdout=subprocess.PIPE if redirect else slave,
                            stderr=slave, cwd=ROOT, preexec_fn=attach)
    os.close(slave)
    transcript = bytearray()
    offset = 0
    deadline = time.monotonic() + 15
    try:
        while True:
            if steps and steps[0][0].encode() in transcript[offset:]:
                marker, answer = steps.pop(0)
                offset = len(transcript)
                os.write(master, answer)
            if time.monotonic() > deadline:
                raise AssertionError("PTY timed out waiting for " + repr(steps[:1]))
            ready, _, _ = select.select([master], [], [], 0.1)
            if ready:
                try:
                    chunk = os.read(master, 65536)
                except OSError as exc:
                    if exc.errno != errno.EIO:
                        raise
                    chunk = b""
                if not chunk:
                    break
                transcript.extend(chunk)
            elif proc.poll() is not None:
                break
        code = proc.wait(timeout=2)
        if redirect:
            transcript.extend(proc.stdout.read())
        if steps:
            raise AssertionError("unreached prompts: " + repr(steps))
        return code, transcript.decode("utf-8", errors="replace")
    finally:
        if proc.poll() is None:
            os.killpg(proc.pid, signal.SIGKILL)
            proc.wait()
        os.close(master)
        if proc.stdout:
            proc.stdout.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    opts = parser.parse_args()
    opts.binary = opts.binary.resolve()
    opts.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    results = []
    with tempfile.TemporaryDirectory(prefix="alm15-pty-") as tmp:
        budget = Path(tmp) / "budget.json"
        budget.write_text(json.dumps(dict(Profile="SYNTHETIC-PTY-NOT-PRODUCTION",
            MinCPUs=1, CPUFlags=["sse2"], MinTotalMemory=1024,
            MinAvailableMemory=512, MinDiskAvailable=4096)))
        base = ["wizard", "--manifest", "manifests/stack.lock.yaml",
                "--requirements", str(budget)]
        loaded = base + ["--config", "internal/config/testdata/basic.json"]
        labels = ["Target server bind IP", "Advertised IP", "Optional hostname",
                  "SIP UDP port", "RTP first UDP port", "RTP last UDP port",
                  "Enable optional WebRTC", "Database credential reference",
                  "AMI credential reference", "SIP credential reference",
                  "Administrator credential reference"]
        normal = [(label, b"\n") for label in labels]
        cases = [
            ("loaded-review", loaded + ["--review-only"], normal.copy(), 80, False, 0,
             ["CANDIDATE", "approval NOT REQUESTED", "NOT STARTED"]),
            ("narrow-candidate-rejection", loaded, normal.copy(), 32, False, 1,
             ["CANDIDATE", "approval_rejected", "Execution: NOT STARTED"]),
            ("fresh-IP-only", base + ["--review-only"],
             [(label, value) for label, value in zip(labels,
              [b"192.0.2.10\n", b"\n", b"\n", b"\n", b"\n", b"\n", b"no\n",
               b"env:DB\n", b"env:AMI\n", b"env:SIP\n", b"env:ADMIN\n"])],
             80, False, 0, ["WebRTC enabled: false", "Target IP: 192.0.2.10"]),
            ("invalid-correction", loaded + ["--review-only"],
             [(labels[0], b"bad\n"), ("Invalid input", b"still-bad\n"),
              ("Invalid input", b"192.0.2.10\n")] + normal[1:],
             80, False, 0, ["Configuration: VALID"]),
            ("EOF", loaded, [(labels[0], b"\x04")], 80, False, 1, ["input ended or failed"]),
            ("explicit-cancel", loaded, [(labels[0], b"cancel\n")], 80, False, 1, ["canceled"]),
            ("Ctrl-C", loaded, [(labels[0], b"partial\x03")], 80, False, 1, ["canceled"]),
            ("redirected-output", loaded, [], 80, True, 2, ["no interactive terminal"]),
        ]
        for name, args, steps, width, redirect, expected, markers in cases:
            code, text = run_pty(opts.binary, args, steps, width, redirect)
            assert code == expected, (name, code, text)
            assert all(marker in text for marker in markers), (name, text)
            assert "\x1b[" not in text, (name, "unexpected ANSI")
            assert "approval_recorded" not in text, (name, "candidate bypass")
            (opts.output / (name + ".txt")).write_text(text)
            results.append(dict(case=name, exit_code=code, status="PASS", width=width))
        for name, extra, expected, marker in [
            ("pipe-rejected", [], 2, "no interactive terminal"),
            ("unattended-review", ["--unattended", "--review-only"], 0, "approval NOT REQUESTED"),
            ("unattended-missing-approval", ["--unattended"], 1, "explicit unattended approval"),
        ]:
            proc = subprocess.run([str(opts.binary), *loaded, *extra], input="yes\n",
                                  text=True, capture_output=True, cwd=ROOT, timeout=15)
            text = proc.stdout + proc.stderr
            assert proc.returncode == expected and marker in text, (name, text)
            assert "Approve this exact plan" not in text
            (opts.output / (name + ".txt")).write_text(text)
            results.append(dict(case=name, exit_code=proc.returncode, status="PASS"))
    (opts.output / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    print(f"PASS: {len(results)} executable terminal/nonterminal journeys")


if __name__ == "__main__":
    main()
