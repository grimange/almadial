import importlib.util
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("prepare", Path(__file__).parents[1] / "prepare_vicidial_r4037.py")
m = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(m)


class PreparationTests(unittest.TestCase):
    def test_unreviewed_and_already_patched_bytes_rejected(self):
        for data in (b"wrong revision", m.WIDE_PS):
            with self.assertRaises(ValueError):
                m.transform("bin/AST_manager_send.pl", data)

    def test_observed_process_correction_only(self):
        data = b"before\n" + m.LEGACY_PS + b"\nafter\n"
        with patch.dict(m.PINS, {"bin/AST_manager_send.pl": m.sha256(data)}):
            self.assertEqual(m.transform("bin/AST_manager_send.pl", data), data.replace(m.LEGACY_PS, m.WIDE_PS))

    def test_sql_privileges_separated_without_schema_rewrite(self):
        data = b"CREATE TABLE test(id INT);\n" + b"\n".join(m.PRIVILEGES) + b"\nUPDATE system_settings SET db_schema_version='1748';"
        with patch.dict(m.PINS, {"extras/MySQL_AST_CREATE_tables.sql": m.sha256(data)}):
            result = m.transform("extras/MySQL_AST_CREATE_tables.sql", data)
        self.assertIn(b"CREATE TABLE test(id INT)", result)
        self.assertIn(b"db_schema_version='1748'", result)
        self.assertNotIn(b"GRANT RELOAD", result)
        self.assertEqual(result.count(b"-- AlmaDial M0-04:"), 3)

    def test_unexpected_occurrences_refused(self):
        with patch.dict(m.PINS, {"bin/AST_manager_send.pl": m.sha256(b"no query")}):
            with self.assertRaises(ValueError):
                m.transform("bin/AST_manager_send.pl", b"no query")

    def test_exclusive_outputs_permissions_and_provenance(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            src, out = root / "src", root / "out"
            (src / "bin").mkdir(parents=True)
            name = "bin/AST_manager_send.pl"
            (src / name).write_bytes(m.LEGACY_PS)
            with patch.object(m, "PINS", {name: m.sha256(m.LEGACY_PS)}):
                ledger = m.prepare(src, out)
                self.assertEqual(ledger["execution"], "NOT_RUN")
                self.assertEqual((out / name).read_bytes(), m.WIDE_PS)
                self.assertEqual(os.stat(out / name).st_mode & 0o777, 0o600)
                self.assertEqual(os.stat(out).st_mode & 0o777, 0o700)
                with self.assertRaises(FileExistsError):
                    m.prepare(src, out)
                (src / name).unlink()
                (src / name).symlink_to(out / name)
                with self.assertRaises(ValueError):
                    m.prepare(src, root / "new")


if __name__ == "__main__":
    unittest.main()
