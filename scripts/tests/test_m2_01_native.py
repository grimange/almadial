"""Real SQLite/ASCII-armor boundary tests, without DNF, root or RPM mutation."""
import ast
from contextlib import closing
import hashlib
import pathlib
import sqlite3
import tempfile
import types
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
ADAPTER = ROOT / "internal/provision/native.py"


def function(name, namespace):
    tree = ast.parse(ADAPTER.read_text())
    node = next(n for n in tree.body if isinstance(n, ast.FunctionDef) and n.name == name)
    exec(compile(ast.Module(body=[node], type_ignores=[]), str(ADAPTER), "exec"), namespace)
    return namespace[name]


class NativeBoundaryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = pathlib.Path(self.temp.name) / "history.sqlite"
        import json
        self.history = function("history", dict(
            pathlib=types.SimpleNamespace(Path=lambda value: self.path if value == "/var/lib/dnf/history.sqlite" else pathlib.Path(value)),
            digest=lambda data: hashlib.sha256(data).hexdigest(), sqlite3=sqlite3, json=json, closing=closing))

    def database(self, rows):
        with closing(sqlite3.connect(self.path)) as db:
            db.execute("CREATE TABLE trans(id INTEGER PRIMARY KEY,dt_begin INTEGER,dt_end INTEGER,state INTEGER)")
            db.executemany("INSERT INTO trans VALUES(?,?,?,?)", rows)
            db.commit()

    def test_absent_history_does_not_create_database(self):
        missing = self.history()
        self.assertFalse(self.path.exists())
        self.database([])
        self.assertEqual(missing, self.history())

    def test_read_only_history_and_semantic_identity(self):
        self.database([(1, 100, 200, 1)])
        original = self.path.read_bytes()
        identity, complete = self.history()
        self.assertTrue(complete)
        self.assertEqual(original, self.path.read_bytes())
        with closing(sqlite3.connect(self.path)) as db:
            db.execute("PRAGMA user_version=9")
        self.assertNotEqual(original, self.path.read_bytes())
        self.assertEqual((identity, True), self.history())

    def test_old_unfinished_transaction_is_not_hidden_by_new_success(self):
        self.database([(1, 100, None, 0), (2, 300, 400, 1)])
        self.assertFalse(self.history()[1])

    def test_invalid_history_fails_closed(self):
        self.path.write_bytes(b"invalid database")
        with self.assertRaises(sqlite3.DatabaseError):
            self.history()

    def test_pinned_public_key_only(self):
        import base64
        seen = []
        ts = types.SimpleNamespace(pgpImportPubkey=lambda payload: seen.append(payload) or 0)
        call = function("import_key", dict(base64=base64, digest=lambda data: hashlib.sha256(data).hexdigest(), rpm=types.SimpleNamespace(TransactionSet=lambda: ts)))
        with self.assertRaises(ValueError):
            call(dict(public_key="untrusted material"))
        self.assertEqual([], seen)
        key = (ROOT / "internal/provision/epel.asc").read_text()
        result = call(dict(public_key=key))
        self.assertEqual("de390fc168eae5ab2852e9e93d34a0b9ddf05cf9ce90ee28d97de26a4b1f6b93", result["key_sha256"])
        self.assertEqual(1, len(seen))
        self.assertGreater(len(seen[0]), 1000)
