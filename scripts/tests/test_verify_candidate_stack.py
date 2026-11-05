"""M0-02 negative-case tests; never contact a VM, DNF, or upstream source."""
import importlib.util
import copy
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / "verify_candidate_stack.py"
SPEC = importlib.util.spec_from_file_location("candidate", SCRIPT)
C = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(C)


class CandidateEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.package = dict(zip(C.FIELDS, ("php-fpm", "0", "8.3.36", "1.el10_2", "x86_64", "appstream")))
        self.baseline = {"enabled_repositories": ["baseos", "appstream", "crb", "extras"],
                         "packages": [{"requested": "php-fpm", "selected": [self.package]}],
                         "missing_in_enabled_repositories": ["perl-Net-Telnet"]}
        self.records = {"os": 'ID="almalinux"\nVERSION_ID="10.2"\n', "architecture": "x86_64\n",
                        "repositories": "repo id repo name\nbaseos BaseOS\nappstream AppStream\ncrb CRB\nextras Extras\n",
                        "packages": "|".join(self.package[k] for k in C.FIELDS) + "\n",
                        "installed-before": "python|0|3.12|1|x86_64\n", "installed-after": "python|0|3.12|1|x86_64\n"}
        self.write_records()

    def write_records(self):
        for name, stdout in self.records.items():
            (self.directory / (name + ".json")).write_text(json.dumps({"exit_code": 0, "stdout": stdout}))

    def test_replay_succeeds_without_live_process(self):
        with patch.object(C.subprocess, "run", side_effect=AssertionError("must not execute")):
            self.assertEqual(C.check_evidence(self.baseline, self.directory), [])

    def test_missing_package_is_explicit(self):
        self.records["packages"] = self.records["packages"].replace("8.3.36", "8.3.35")
        self.write_records()
        self.assertIn("pinned RPM unavailable: php-fpm", C.check_evidence(self.baseline, self.directory))

    def test_newer_version_does_not_hide_available_pin(self):
        self.records["packages"] += self.records["packages"].replace("8.3.36", "8.3.37")
        self.write_records()
        self.assertEqual(C.check_evidence(self.baseline, self.directory), [])

    def test_empty_metadata_is_not_success(self):
        self.records["packages"] = ""
        self.write_records()
        self.assertIn("empty package metadata result", C.check_evidence(self.baseline, self.directory))

    def test_wrong_os_and_repository_reported(self):
        self.records["os"] = 'ID=ubuntu\nVERSION_ID="24.04"\n'
        self.records["repositories"] = "baseos BaseOS\n"
        self.write_records()
        issues = C.check_evidence(self.baseline, self.directory)
        self.assertIn("OS mismatch: ID", issues)
        self.assertIn("repository unavailable: appstream", issues)

    def test_command_failure_not_interpreted_as_missing(self):
        (self.directory / "packages.json").write_text(json.dumps({"exit_code": 1, "stdout": ""}))
        with self.assertRaisesRegex(ValueError, "command failed"):
            C.check_evidence(self.baseline, self.directory)

    def test_installed_package_mutation_detected(self):
        self.records["installed-after"] += "unexpected|0|1|1|noarch\n"
        self.write_records()
        self.assertIn("installed RPM set changed during qualification", C.check_evidence(self.baseline, self.directory))

    def test_malformed_package_output(self):
        for line in ("DNF failed", "php|unknown|8|1|x86_64|appstream", "php|0|8|1|aarch64|appstream"):
            with self.subTest(line=line), self.assertRaises(ValueError):
                C.parse_packages(line)

    def test_source_tampering_and_tree_digest(self):
        path = self.directory / "file"
        path.write_text("original")
        digest = C.tree_digest(self.directory)
        path.write_text("changed")
        self.assertNotEqual(C.tree_digest(self.directory), digest)
        baseline = {"sources": [{"component": "test", "artifact": "file", "sha256": "0" * 64}]}
        self.assertEqual(C.check_sources(baseline, self.directory), ["source digest mismatch: test"])

    def test_tree_rejects_symlinks(self):
        (self.directory / "link").symlink_to("/etc/passwd")
        with self.assertRaisesRegex(ValueError, "symlink"):
            C.tree_digest(self.directory)

    def test_manifest_drift_and_promotion_detected(self):
        baseline = json.loads(C.BASELINE.read_text())
        manifest = (C.ROOT / "manifests/stack.lock.yaml").read_text()
        self.assertEqual(C.check_local(baseline, manifest), [])
        self.assertIn("manifest mismatch: status", C.check_local(baseline, manifest.replace("status: candidate", "status: accepted")))

    def test_matrix_mapping_and_state_checked(self):
        baseline = json.loads(C.BASELINE.read_text())
        text = (C.ROOT / "docs/compatibility/m0-02-dependency-matrix.md").read_text()
        self.assertEqual(C.check_matrix(baseline, text), [])
        self.assertTrue(C.check_matrix(baseline, text.replace("| Not Tested |", "| PASS |", 1)))

    def test_m004_provenance_transition_requires_exact_archive_and_tree(self):
        baseline = json.loads(C.BASELINE.read_text())
        manifest = (C.ROOT / "manifests/stack.lock.yaml").read_text()
        self.assertEqual(C.check_local(baseline, manifest), [])
        for key, value in C.M004_PROVENANCE.items():
            with self.subTest(key=key):
                changed = manifest.replace(value, "unverified-value")
                self.assertIn("manifest mismatch: vicidial.provenance_state",
                              C.check_local(baseline, changed))

    def test_historical_m002_provenance_remains_valid(self):
        baseline = json.loads(C.BASELINE.read_text())
        manifest = (C.ROOT / "manifests/stack.lock.yaml").read_text()
        manifest = manifest.replace(C.M004_PROVENANCE["vicidial.provenance_state"],
                                    baseline["manifest_pins"]["vicidial.provenance_state"])
        self.assertEqual(C.check_local(baseline, manifest), [])

    def test_source_records_must_agree_with_manifest(self):
        baseline = json.loads(C.BASELINE.read_text())
        manifest = (C.ROOT / "manifests/stack.lock.yaml").read_text()
        for field, value in (("version", "0.0.0"), ("sha256", "0" * 64), ("url", "https://example.invalid/wrong")):
            with self.subTest(field=field):
                changed = copy.deepcopy(baseline)
                changed["sources"][0][field] = value
                self.assertTrue(C.check_local(changed, manifest))

    def test_required_source_cannot_be_missing_or_duplicated(self):
        baseline = json.loads(C.BASELINE.read_text())
        manifest = (C.ROOT / "manifests/stack.lock.yaml").read_text()
        for sources in (baseline["sources"][1:], baseline["sources"] + [baseline["sources"][0]]):
            changed = copy.deepcopy(baseline)
            changed["sources"] = sources
            self.assertIn("source identity missing or duplicated: Asterisk", C.check_local(changed, manifest))

    def test_selected_runtime_package_must_agree_with_manifest(self):
        baseline = json.loads(C.BASELINE.read_text())
        manifest = (C.ROOT / "manifests/stack.lock.yaml").read_text()
        row = next(row for row in baseline["packages"] if row["requested"] == "php-fpm")
        row["selected"][0]["version"] = "0.0.0"
        self.assertIn("package/manifest identity mismatch: php", C.check_local(baseline, manifest))

    def test_duplicate_manifest_container_rejected(self):
        with self.assertRaisesRegex(ValueError, "duplicate manifest key"):
            C.manifest_leaves("asterisk:\n  version: 22.12.0\nasterisk:\n  sha256: other\n")

    def test_existing_output_never_overwritten(self):
        with patch.object(C, "run_record", side_effect=AssertionError("must not execute")):
            with self.assertRaises(FileExistsError):
                C.collect(self.baseline, self.directory, False)

    def test_live_collection_only_uses_named_vm_and_read_commands(self):
        output = self.directory / "new"
        with patch.object(C, "run_record", return_value={"exit_code": 0, "stdout": ""}) as run:
            C.collect(self.baseline, output, False)
        for call in run.call_args_list:
            argv = call.args[0]
            self.assertEqual(argv[:4], ["multipass", "exec", "almadial-alma10", "--"])
            self.assertFalse(set(argv) & {"install", "remove", "upgrade", "makecache"})
        self.assertEqual(output.stat().st_mode & 0o777, 0o700)

    def test_missing_command_and_timeout_classified(self):
        for exception, state in ((FileNotFoundError(), "UNAVAILABLE"), (subprocess.TimeoutExpired(["test"], 120), "TIMEOUT")):
            with self.subTest(state=state), patch.object(C.subprocess, "run", side_effect=exception):
                record = C.run_record(["test"])
                self.assertIsNone(record["exit_code"])
                self.assertEqual(record["error"], state)


if __name__ == "__main__":
    unittest.main()
