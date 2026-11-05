"""Offline M0 collector contract tests; no inventory commands run on this host."""
import copy
import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import collect_host_inventory as c

FIXTURE = Path(__file__).parent / "fixtures/inventory/almalinux10.json"


class InventoryTests(unittest.TestCase):
    def setUp(self):
        self.fixture = json.loads(FIXTURE.read_text())
        self.r = c.Redactor(b"fixture-only-key")

    def test_complete_fixture_is_not_host_acceptance(self):
        with patch.object(c, "live_probe", side_effect=AssertionError("live probe forbidden")):
            doc = c.inventory(self.fixture)
        self.assertEqual(doc["mode"], "FIXTURE")
        self.assertEqual(doc["schema_version"], 1)
        self.assertEqual(doc["host_baseline_acceptance"], "NOT_RUN")
        self.assertEqual(doc["compatibility"], "NOT_RUN")
        self.assertEqual(doc["independent_qa"], "NOT_RUN")
        self.assertTrue(doc["started_at_utc"].endswith("Z"))
        self.assertEqual([x["id"] for x in doc["probes"]], list(c.PROBES))
        self.assertEqual({x["status"] for x in doc["probes"]}, {"COLLECTED"})
        self.assertTrue(all(x["simulated"] for x in doc["probes"]))
        probes = {x["id"]: x for x in doc["probes"]}
        self.assertEqual(probes["os"]["facts"]["version"], "10.0")
        self.assertEqual(probes["cpu"]["facts"]["cpu_count"], 4)
        self.assertIn("avx2", probes["cpu"]["facts"]["flags"])
        self.assertEqual(probes["memory"]["facts"]["MemTotal_bytes"], 8589934592)
        self.assertIn("asterisk", probes["packages"]["facts"]["not_installed"])
        self.assertEqual(probes["packages"]["exit_code"], 7)
        self.assertFalse(probes["repositories"]["facts"]["repositories"][1]["configured_enabled"])
        self.assertEqual(probes["interfaces"]["facts"][0]["addresses"][1]["family"], 6)
        self.assertEqual(probes["listeners"]["facts"][0]["port"], 22)

    def test_sensitive_input_never_exported(self):
        self.fixture["probes"]["kernel"]["stderr"] = "DOC_ONLY_SECRET secret@example.test /home/private"
        doc = c.inventory(self.fixture, {"host_ref": "actual-private-host.example.test"})
        encoded = json.dumps(doc) + c.summary(doc)
        for forbidden in ("DOC_ONLY_SECRET", "192.0.2.", "2001:db8", "example.test", "fixture-user", "00:11:22:33:44:55", "/home/fixture-private", "ens3", "baseos"):
            self.assertNotIn(forbidden, encoded)
        self.assertTrue(next(x for x in doc["probes"] if x["id"] == "kernel")["stderr_present"])
        self.assertIn("avx2", encoded)
        self.assertIn("Enforcing", encoded)

    def test_redaction_is_consistent_within_run_not_across_runs(self):
        self.assertEqual(self.r.token("192.0.2.10"), self.r.token("192.0.2.10"))
        self.assertNotEqual(self.r.token("192.0.2.10"), c.Redactor().token("192.0.2.10"))

    def test_missing_fixture_probe_never_falls_back_to_host(self):
        del self.fixture["probes"]["cpu"]
        with patch.object(c, "live_probe", side_effect=AssertionError("no fallback")):
            doc = c.inventory(self.fixture)
        row = next(x for x in doc["probes"] if x["id"] == "cpu")
        self.assertEqual(row["status"], "NOT_RUN")
        self.assertIsNone(row["facts"])

    def test_errors_missing_commands_and_malformed_output(self):
        cases = [
            ({"error": "command_not_found"}, "UNAVAILABLE", "command_not_found"),
            ({"error": "permission_denied"}, "ERROR", "permission_denied"),
            ({"error": "timeout", "exit_code": -9}, "ERROR", "timeout"),
            ({"exit_code": 1, "stdout": "secret", "stderr": "private"}, "ERROR", "nonzero_exit"),
            ({"stdout": "6.12.0"}, "ERROR", "missing_exit_code"),
            ({"exit_code": 0, "stdout": "not a version"}, "ERROR", "parse_error"),
            ({"exit_code": "0"}, "ERROR", "invalid_exit_code"),
            ({"exit_code": 0, "stdout": {}}, "ERROR", "invalid_output"),
        ]
        for raw, status, error in cases:
            with self.subTest(raw=raw):
                row = c.result("kernel", raw, self.r, True)
                self.assertEqual((row["status"], row["error"]), (status, error))
                self.assertNotIn("secret", json.dumps(row))
                self.assertIsNone(row["facts"])

    def test_malformed_structured_output_is_error(self):
        for probe, text in [("cpu", "{bad"), ("interfaces", "{}"),
                            ("listeners", "unexpected output"), ("services", "Id=unknown"),
                            ("packages", "package manager failed")]:
            with self.subTest(probe=probe):
                self.assertEqual(c.result(probe, {"stdout": text, "exit_code": 0}, self.r, True)["status"], "ERROR")

    def test_incomplete_required_facts_are_partial_not_collected(self):
        cases = [
            ("memory", "MemTotal:       8388608 kB\n"),
            ("time", "Timezone=\nNTPSynchronized=unexpected\nLocalRTC=unexpected\n"),
            ("time", "Timezone=Etc/UTC\nNTPSynchronized=unexpected\nLocalRTC=no\n"),
            ("chrony", "Stratum : 3\n"),
            ("services", "\n\n".join("Id=" + u for u in c.UNITS)),
            ("disks", '{"blockdevices":[{}]}'),
            ("mounts", '{"filesystems":[{}]}'),
            ("interfaces", '[{"ifname":"ens3"}]'),
            ("routes_v4", '[{}]'),
            ("routes_v6", '[{"dst":"default"}]'),
        ]
        for probe, text in cases:
            with self.subTest(probe=probe, text=text):
                fixture = copy.deepcopy(self.fixture)
                fixture["probes"][probe]["stdout"] = text
                with tempfile.TemporaryDirectory() as tmp, patch.object(c, "live_probe", side_effect=AssertionError("no host execution")):
                    source = Path(tmp) / "fixture.json"
                    source.write_text(json.dumps(fixture))
                    target = Path(tmp) / "evidence"
                    self.assertEqual(c.main(["--fixture", str(source), "--output-dir", str(target)]), 1)
                    doc = json.loads((target / "inventory.json").read_text())
                    row = next(x for x in doc["probes"] if x["id"] == probe)
                    self.assertEqual((row["status"], row["error"], row["facts"]), ("ERROR", "parse_error", None))
                    self.assertIn(f"| {probe} | ERROR |", (target / "summary.md").read_text())

    def test_invalid_required_metrics_and_states_are_errors(self):
        cases = [
            ("chrony", self.fixture["probes"]["chrony"]["stdout"].replace("Normal", "unexpected")),
            ("chrony", self.fixture["probes"]["chrony"]["stdout"].replace("Stratum         : 3", "Stratum         : 99")),
            ("services", self.fixture["probes"]["services"]["stdout"].replace("ActiveState=active", "ActiveState=unexpected")),
            ("disks", '{"blockdevices":[{"name":"vda","type":"disk","size":"bad","fstype":null,"mountpoints":[null]}]}'),
            ("disks", '{"blockdevices":[{"name":"vda","type":"disk","size":10,"fstype":null,"mountpoints":"/private"}]}'),
            ("interfaces", self.fixture["probes"]["interfaces"]["stdout"].replace('"prefixlen": 24', '"prefixlen": 99')),
        ]
        for probe, text in cases:
            with self.subTest(probe=probe, text=text):
                row = c.result(probe, {"stdout": text, "exit_code": 0}, self.r, True)
                self.assertEqual((row["status"], row["error"]), ("ERROR", "parse_error"))

    def test_valid_absent_and_nullable_facts_remain_distinct(self):
        doc = c.inventory(self.fixture)
        rows = {x["id"]: x for x in doc["probes"]}
        self.assertEqual(rows["services"]["facts"][0]["enabled"], "not_applicable")
        self.assertEqual(rows["services"]["facts"][0]["load"], "not-found")
        self.assertEqual(rows["disks"]["facts"][0]["filesystem"], "unknown")
        self.assertEqual(rows["chrony"]["facts"]["System time direction"], "slow")
        raw = {"stdout": '{"filesystems":[{"source":"proc","target":"/proc","size":null,"avail":null,"fstype":"proc"}]}', "exit_code": 0}
        row = c.result("mounts", raw, self.r, True)
        self.assertEqual(row["status"], "COLLECTED")
        self.assertIsNone(row["facts"][0]["size_bytes"])
        raw = {"stdout": '[{"dst":"192.0.2.0/24","type":"blackhole"}]', "exit_code": 0}
        row = c.result("routes_v4", raw, self.r, True)
        self.assertEqual(row["status"], "COLLECTED")
        self.assertIsNone(row["facts"][0]["interface"])

    def test_failed_os_gate_does_not_run_other_host_probes(self):
        with patch.object(c, "live_probe", return_value={"stdout": 'ID=ubuntu\nVERSION_ID="24.04"', "file_read": True}) as live:
            doc = c.inventory()
        live.assert_called_once_with("os")
        self.assertEqual(doc["probes"][0]["status"], "ERROR")
        self.assertTrue(all(x["status"] == "NOT_RUN" for x in doc["probes"][1:]))

    def test_absent_rpm_result_must_be_complete(self):
        raw = copy.deepcopy(self.fixture["probes"]["packages"])
        raw["stdout"] = "package asterisk is not installed\n"
        self.assertEqual(c.result("packages", raw, self.r, True)["status"], "ERROR")

    def test_rpm_multiple_absent_packages_on_almalinux_10(self):
        raw = copy.deepcopy(self.fixture["probes"]["packages"])
        raw["stdout"] = raw["stdout"].replace("firewalld\t0:10.0-1.el10\tx86_64", "package firewalld is not installed")
        raw["exit_code"] = 8
        row = c.result("packages", raw, self.r, True)
        self.assertEqual(row["status"], "COLLECTED")
        self.assertEqual(len(row["facts"]["not_installed"]), 8)
        for code in (1, 9, 127, -9):
            raw["exit_code"] = code
            self.assertEqual(c.result("packages", raw, self.r, True)["error"], "nonzero_exit")

    def test_timedatectl_repeats_property_flags(self):
        self.assertEqual(c.COMMANDS["time"], ("timedatectl", "show",
                         "--property=NTPSynchronized", "--property=Timezone", "--property=LocalRTC"))
        # AlmaLinux 10.2 returned exit 0 with empty output for one comma-list.
        self.assertEqual(c.result("time", {"stdout": "", "exit_code": 0}, self.r, True)["error"], "parse_error")
        raw = {"stdout": "Timezone=Etc/UTC\nLocalRTC=no\nNTPSynchronized=yes\n", "exit_code": 0}
        self.assertEqual(c.result("time", raw, self.r, True)["facts"]["ntp_synchronized"], "yes")

    def test_existing_services_and_unknown_units(self):
        raw = copy.deepcopy(self.fixture["probes"]["services"])
        raw["stdout"] = raw["stdout"].replace("Id=asterisk.service\nLoadState=not-found\nActiveState=inactive\nSubState=dead\nUnitFileState=", "Id=asterisk.service\nLoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled")
        row = c.result("services", raw, self.r, True)
        self.assertEqual(row["facts"][0]["active"], "active")

    def test_no_sudo_shell_install_or_network_commands_in_catalog(self):
        for argv in c.COMMANDS.values():
            self.assertNotIn(argv[0], {"sh", "bash", "sudo", "dnf", "curl", "wget", "ssh"})
            self.assertNotIn("--reload", argv)


class RunnerAndCliTests(unittest.TestCase):
    def test_runner_bounds_and_exit_codes_without_host_inventory(self):
        # These subprocesses only emit synthetic bytes; no host probes execute.
        row = c.run_command((sys.executable, "-c", "import sys;print('fixture');print('synthetic error',file=sys.stderr);sys.exit(3)"))
        self.assertEqual(row["exit_code"], 3)
        self.assertIn("synthetic error", row["stderr"])
        row = c.run_command((sys.executable, "-c", "import time;time.sleep(2)"), timeout=0.05)
        self.assertEqual(row["error"], "timeout")
        row = c.run_command((sys.executable, "-c", "print('x'*100000)"), limit=1024)
        self.assertEqual(row["error"], "output_limit")
        self.assertLessEqual(len(row["stdout"]) + len(row["stderr"]), 1024)
        self.assertEqual(c.run_command(("/nonexistent-documentation-only/command",))["error"], "command_not_found")

    def test_cli_requires_explicit_mode_and_authority_references(self):
        with patch.object(c, "inventory", side_effect=AssertionError("no host execution")), contextlib.redirect_stderr(io.StringIO()):
            for args in ([], ["--collect", "--output-dir", "/tmp/unused-inventory-test"]):
                with self.assertRaises(SystemExit) as raised:
                    c.main(args)
                self.assertEqual(raised.exception.code, 2)

    def test_cli_offline_outputs_permissions_and_no_overwrite(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp) / "evidence"
            args = ["--fixture", str(FIXTURE), "--output-dir", str(target)]
            with patch.object(c, "live_probe", side_effect=AssertionError("no host access")):
                self.assertEqual(c.main(args), 0)
            doc = json.loads((target / "inventory.json").read_text())
            self.assertEqual(doc["mode"], "FIXTURE")
            self.assertIn("Sanitized facts", (target / "summary.md").read_text())
            self.assertEqual(os.stat(target).st_mode & 0o777, 0o700)
            self.assertEqual(os.stat(target / "inventory.json").st_mode & 0o777, 0o600)
            with self.assertRaises(SystemExit), contextlib.redirect_stderr(io.StringIO()):
                c.main(args)

    def test_cli_invalid_fixture_and_partial_result_exit_codes(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "fixture.json"
            path.write_text('{"fixture_schema":1,"probes":{}}')
            self.assertEqual(c.main(["--fixture", str(path), "--output-dir", str(Path(tmp)/"partial")]), 1)
            path.write_text('{"fixture_schema":1,"probes":{"bogus":{}}}')
            with contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(c.main(["--fixture", str(path), "--output-dir", str(Path(tmp)/"bad")]), 2)
            self.assertFalse((Path(tmp)/"bad").exists())

    def test_null_and_wrong_fixture_shapes_never_invoke_inventory(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "fixture.json"
            with patch.object(c, "inventory", side_effect=AssertionError("no host execution")), contextlib.redirect_stderr(io.StringIO()):
                for value in (None, [], 1, {"fixture_schema": True, "probes": {}}, {"fixture_schema": 1, "probes": []}):
                    with self.subTest(value=value):
                        path.write_text(json.dumps(value))
                        self.assertEqual(c.main(["--fixture", str(path), "--output-dir", str(Path(tmp)/"bad")]), 2)


if __name__ == "__main__":
    unittest.main()
