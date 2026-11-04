"""Unit tests for the documentation scaffold validator (not an installer test)."""
import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from validate_scaffold import parse_frontmatter, validate_tree  # noqa: E402


class FrontmatterTests(unittest.TestCase):
    def test_valid_skill(self):
        result = parse_frontmatter("---\nname: test-skill\ndescription: Do a well-scoped thing when the matching task is requested.\n---\n# Instructions\n")
        self.assertEqual(result["name"], "test-skill")

    def test_missing_description_rejected(self):
        with self.assertRaises(ValueError):
            parse_frontmatter("---\nname: test-skill\n---\n# Instructions\n")

    def test_invalid_name_rejected(self):
        with self.assertRaises(ValueError):
            parse_frontmatter("---\nname: BadName\ndescription: Do a well-scoped thing when the matching task is requested.\n---\n")

    def test_missing_tree_fails(self):
        with tempfile.TemporaryDirectory() as d:
            self.assertIn("missing file: AGENTS.md", validate_tree(Path(d)))


if __name__ == "__main__":
    unittest.main()
