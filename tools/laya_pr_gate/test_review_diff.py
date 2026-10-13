"""Regression checks for the advisory review's content boundary."""

import contextlib
import importlib.util
import io
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("review_diff", Path(__file__).with_name("review_diff.py"))
review = importlib.util.module_from_spec(spec)
spec.loader.exec_module(review)


class ReviewBoundaryTests(unittest.TestCase):
    def test_environment_and_key_paths_are_excluded_before_content_read(self):
        for path in [".env", ".env.local", "apps/web/.env.example", "credentials/a.json", "a/private.pem", "../outside"]:
            self.assertFalse(review.eligible_path(path), path)
        with patch.object(review, "git_output", return_value="apps/web/.env.local\napps/web/page.tsx\n"):
            self.assertEqual(review.changed_paths("HEAD~1"), ["apps/web/page.tsx"])

    def test_symlink_content_is_never_read(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "source.txt").write_text("synthetic sentinel", encoding="utf-8")
            (root / "link.txt").symlink_to(root / "source.txt")
            with patch.object(review, "ROOT", root), patch.object(review, "git_output", return_value=""):
                self.assertEqual(review.diff_text(None, "link.txt"), "")

    def test_signals_do_not_mix_documentation_and_code_or_echo_content(self):
        output = io.StringIO()
        paths = ["docs/guide.md", "apps/web/page.tsx"]
        with patch.object(review, "changed_paths", return_value=paths), patch.object(
            review, "diff_text", side_effect=["+localStorage.setItem('token', 'synthetic-sentinel')", "+const x = 1;"]
        ), patch("sys.argv", ["review_diff.py"]), contextlib.redirect_stdout(output):
            self.assertEqual(review.main(), 0)
        self.assertIn("Sinais automáticos: nenhum.", output.getvalue())
        self.assertNotIn("synthetic-sentinel", output.getvalue())

    def test_git_failure_remains_advisory_and_sanitized(self):
        output = io.StringIO()
        with patch.object(review, "changed_paths", side_effect=RuntimeError("synthetic-sentinel")), patch(
            "sys.argv", ["review_diff.py"]
        ), contextlib.redirect_stderr(output):
            self.assertEqual(review.main(), 0)
        self.assertNotIn("synthetic-sentinel", output.getvalue())


if __name__ == "__main__":
    unittest.main()
