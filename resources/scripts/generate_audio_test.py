"""
Tests for generate_audio.py — written before implementation (TDD).

Run from the resources/ directory:
    cd resources && pytest scripts/generate_audio_test.py -v
"""
import hashlib
import json
import os
import unicodedata
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

import sys
sys.path.insert(0, str(Path(__file__).parent))
from generate_audio import audio_filename, generate_audio, main  # noqa: E402

FIXTURES_DIR = Path(__file__).parent.parent / "fixtures"


def load_fixture(name: str):
    return json.loads((FIXTURES_DIR / name).read_text(encoding="utf-8"))


def _mock_client(audio_bytes: bytes = b"fake-ogg"):
    """Return a (mock_class, mock_instance) pair that injects a fixed audio response."""
    mock_resp = MagicMock()
    mock_resp.audio_content = audio_bytes

    mock_instance = MagicMock()
    mock_instance.synthesize_speech.return_value = mock_resp

    mock_class = MagicMock(return_value=mock_instance)
    return mock_class, mock_instance


# ---------------------------------------------------------------------------
# TestAudioFilename — cross-language contract
# ---------------------------------------------------------------------------
class TestAudioFilename:
    """audio_filename() must produce the same hashes as Go's audioFilename()."""

    def test_known_values(self):
        """Each case from the committed fixture must match."""
        for case in load_fixture("audio_filename_cases.json"):
            expected = case["expected"]
            if "nfd_input_hex" in case:
                text = bytes.fromhex(case["nfd_input_hex"]).decode("utf-8")
            else:
                text = case["input"]
            assert audio_filename(text) == expected, (
                f"Fixture {case['note']!r}: "
                f"audio_filename({text!r}) = {audio_filename(text)!r}, "
                f"want {expected!r}"
            )

    def test_nfc_vs_nfd_equivalence(self):
        """NFD-encoded text must hash the same as NFC."""
        for word in ["üben", "über", "schöne", "Züge", "Mädchen"]:
            nfc = unicodedata.normalize("NFC", word)
            nfd = unicodedata.normalize("NFD", word)
            assert audio_filename(nfd) == audio_filename(nfc), (
                f"NFC/NFD mismatch for {word!r}"
            )

    def test_case_insensitive(self):
        assert audio_filename("LERNEN") == audio_filename("lernen")
        assert audio_filename("Lernen") == audio_filename("lernen")

    def test_whitespace_stripped(self):
        assert audio_filename("  lernen  ") == audio_filename("lernen")
        assert audio_filename("\tlernen\n") == audio_filename("lernen")

    def test_output_format(self):
        for word in ["lernen", "üben", "sein", "arbeiten"]:
            result = audio_filename(word)
            assert result.endswith(".ogg"), f"{result!r} does not end in .ogg"
            assert len(result) == 68, f"len={len(result)}, want 68"

    def test_csv_umlaut_characters(self):
        for char in ["ä", "ö", "ü", "ß", "Ä", "Ö", "Ü"]:
            result = audio_filename(char)
            assert len(result) == 68
            assert result.endswith(".ogg")

    def test_matches_python_reference_implementation(self):
        def ref(text: str) -> str:
            norm = unicodedata.normalize("NFC", text.strip().lower())
            return hashlib.sha256(norm.encode("utf-8")).hexdigest() + ".ogg"

        for word in ["lernen", "üben", "sein", "  LERNEN  ", "arbeiten"]:
            assert audio_filename(word) == ref(word)


# ---------------------------------------------------------------------------
# TestGenerateAudio — core synthesis logic
# ---------------------------------------------------------------------------
class TestGenerateAudio:
    """Patches generate_audio.TextToSpeechClient to avoid any credential lookup."""

    def test_file_written_on_success(self, tmp_path):
        """When SDK returns bytes, the .ogg file is created at the expected path."""
        from google.cloud.texttospeech import AudioEncoding

        mock_class, mock_instance = _mock_client(b"lernen-audio")
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen"], tmp_path, voice="de-DE-Neural2-F")

        mock_instance.synthesize_speech.assert_called_once()
        req = mock_instance.synthesize_speech.call_args.kwargs.get("request") or \
              mock_instance.synthesize_speech.call_args.args[0]
        assert req["voice"]["language_code"] == "de-DE"
        assert req["voice"]["name"] == "de-DE-Neural2-F"
        assert req["audio_config"]["audio_encoding"] == AudioEncoding.OGG_OPUS

        expected = tmp_path / audio_filename("lernen")
        assert expected.exists()
        assert expected.read_bytes() == b"lernen-audio"

    def test_correct_text_sent_to_api(self, tmp_path):
        """The German word is sent as synthesis input text."""
        mock_class, mock_instance = _mock_client()
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen"], tmp_path)

        req = mock_instance.synthesize_speech.call_args.kwargs.get("request") or \
              mock_instance.synthesize_speech.call_args.args[0]
        assert req["input"]["text"] == "lernen"

    def test_deduplication(self, tmp_path):
        """Duplicate words (after normalization) → only one API call."""
        mock_class, mock_instance = _mock_client()
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen", "lernen", "LERNEN", "  lernen  "], tmp_path)

        assert mock_instance.synthesize_speech.call_count == 1

    def test_multiple_words(self, tmp_path):
        """Each unique word produces a separate file."""
        def side_effect(**kwargs):
            resp = MagicMock()
            resp.audio_content = kwargs["request"]["input"]["text"].encode()
            return resp

        mock_instance = MagicMock()
        mock_instance.synthesize_speech.side_effect = side_effect
        mock_class = MagicMock(return_value=mock_instance)

        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen", "sein", "arbeiten"], tmp_path)

        for word in ["lernen", "sein", "arbeiten"]:
            assert (tmp_path / audio_filename(word)).exists()

    def test_output_dir_created(self, tmp_path):
        """generate_audio creates the output directory if it doesn't exist."""
        nested = tmp_path / "a" / "b" / "c"
        mock_class, _ = _mock_client()
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen"], nested)

        assert nested.is_dir()


# ---------------------------------------------------------------------------
# TestIdempotency
# ---------------------------------------------------------------------------
class TestIdempotency:
    def test_existing_file_skipped(self, tmp_path):
        """Pre-created .ogg file → SDK is never called."""
        existing = tmp_path / audio_filename("lernen")
        existing.write_bytes(b"pre-existing")

        mock_class, mock_instance = _mock_client()
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen"], tmp_path)

        mock_instance.synthesize_speech.assert_not_called()
        assert existing.read_bytes() == b"pre-existing"

    def test_force_flag_regenerates(self, tmp_path):
        """force=True overwrites existing files."""
        existing = tmp_path / audio_filename("lernen")
        existing.write_bytes(b"old-bytes")

        mock_class, mock_instance = _mock_client(b"new-bytes")
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen"], tmp_path, force=True)

        mock_instance.synthesize_speech.assert_called_once()
        assert existing.read_bytes() == b"new-bytes"

    def test_second_run_is_noop(self, tmp_path):
        """Running generate_audio twice → second run makes zero API calls."""
        mock_class, mock_instance = _mock_client(b"audio")
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen"], tmp_path)
            assert mock_instance.synthesize_speech.call_count == 1
            generate_audio(["lernen"], tmp_path)
            assert mock_instance.synthesize_speech.call_count == 1


# ---------------------------------------------------------------------------
# TestDryRun
# ---------------------------------------------------------------------------
class TestDryRun:
    def test_no_files_written(self, tmp_path):
        """dry_run=True: no API calls, no files."""
        mock_class, mock_instance = _mock_client()
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen", "sein"], tmp_path, dry_run=True)

        mock_instance.synthesize_speech.assert_not_called()
        assert list(tmp_path.iterdir()) == []

    def test_dry_run_does_not_create_output_dir(self, tmp_path):
        """dry_run=True: output directory is not created if it doesn't exist."""
        nonexistent = tmp_path / "audio"
        mock_class, _ = _mock_client()
        with patch("generate_audio.TextToSpeechClient", mock_class):
            generate_audio(["lernen"], nonexistent, dry_run=True)

        assert not nonexistent.exists()


# ---------------------------------------------------------------------------
# TestMissingCredentials
# ---------------------------------------------------------------------------
class TestMissingCredentials:
    def test_clean_error_on_missing_credentials(self, tmp_path, capsys):
        """Missing credentials → exit code 2, GOOGLE_APPLICATION_CREDENTIALS in stderr."""
        from google.auth.exceptions import DefaultCredentialsError

        env_without_creds = {k: v for k, v in os.environ.items()
                             if k != "GOOGLE_APPLICATION_CREDENTIALS"}

        with patch.dict(os.environ, env_without_creds, clear=True):
            with patch(
                "generate_audio.TextToSpeechClient",
                side_effect=DefaultCredentialsError("no credentials"),
            ):
                with pytest.raises(SystemExit) as exc:
                    main([str(Path(__file__).parent.parent / "Deutch.csv"), str(tmp_path)])
                assert exc.value.code == 2

        stderr = capsys.readouterr().err
        assert "GOOGLE_APPLICATION_CREDENTIALS" in stderr
