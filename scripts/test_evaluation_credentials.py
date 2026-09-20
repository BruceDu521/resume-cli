import tempfile
import unittest
from pathlib import Path
from evaluation_credentials import load_credentials


class CredentialsTest(unittest.TestCase):
    def test_single_provider_shared_key(self):
        self.assertEqual(load_credentials(["deepseek", "deepseek"], environ={"RESUME_AI_API_KEY": "fake"}), {"deepseek": "fake"})
        self.assertEqual(load_credentials([""], environ={}), {})

    def test_never_send_one_key_to_multiple_vendors(self):
        with self.assertRaisesRegex(ValueError, "multiple providers"):
            load_credentials(["deepseek", "gemini"], environ={"RESUME_AI_API_KEY": "fake"})
        with self.assertRaisesRegex(ValueError, "missing RESUME_AI_API_KEY"):
            load_credentials(["deepseek"], environ={"DEEPSEEK_API_KEY": "legacy"})

    def test_explicit_profiles(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "deepseek.env").write_text('# private\nRESUME_AI_API_KEY="ds-fake"\n')
            (root / "gemini.env").write_text("RESUME_AI_API_KEY='gem-fake'\n")
            self.assertEqual(load_credentials(["deepseek", "gemini"], root), {"deepseek": "ds-fake", "gemini": "gem-fake"})
            with self.assertRaisesRegex(ValueError, "cannot read"):
                load_credentials(["kimi"], root)

    def test_bad_profiles_never_echo_contents(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for content in ["SECRET=do-not-echo", "RESUME_AI_API_KEY=\n", "RESUME_AI_API_KEY=a\nRESUME_AI_API_KEY=do-not-echo\n"]:
                (root / "deepseek.env").write_text(content)
                with self.assertRaises(ValueError) as exc:
                    load_credentials(["deepseek"], root)
                self.assertNotIn("do-not-echo", str(exc.exception))


if __name__ == "__main__":
    unittest.main()
