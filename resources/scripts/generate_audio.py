"""
generate_audio.py — synthesize OGG/Opus pronunciation files via Google Cloud TTS.

Usage:
    python3 resources/scripts/generate_audio.py resources/Deutch.csv resources/audio/
    python3 resources/scripts/generate_audio.py resources/Deutch.csv resources/audio/ --dry-run
    python3 resources/scripts/generate_audio.py resources/Deutch.csv resources/audio/ --force

Requires:
    GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

The output filename for each word is deterministic:
    SHA256(NFC(lowercase(strip(german_text)))).hexdigest() + ".ogg"
This is the cross-language contract shared with Go's seed_audio.go::audioFilename().
"""
import argparse
import csv
import hashlib
import logging
import sys
import time
import unicodedata
from pathlib import Path

from google.auth.exceptions import DefaultCredentialsError
from google.cloud.texttospeech import (
    AudioConfig,
    AudioEncoding,
    SynthesisInput,
    TextToSpeechClient,
    VoiceSelectionParams,
)

logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
log = logging.getLogger(__name__)


def audio_filename(text: str) -> str:
    """Cross-language contract — must match Go's seed_audio.go::audioFilename() exactly.

    Algorithm: NFC-normalize → lowercase → strip → SHA256 → hex + ".ogg"
    """
    normalized = unicodedata.normalize("NFC", text.strip().lower())
    return hashlib.sha256(normalized.encode("utf-8")).hexdigest() + ".ogg"


def generate_audio(
    words: list[str],
    output_dir: Path,
    *,
    voice: str = "de-DE-Neural2-F",
    dry_run: bool = False,
    force: bool = False,
    request_interval: float = 1.0,
) -> None:
    """Synthesize OGG/Opus audio for each unique German word.

    Args:
        words: German words to synthesize (duplicates are deduplicated).
        output_dir: Directory where .ogg files are written.
        voice: Google Cloud TTS voice name.
        dry_run: Log what would be done without calling the API or writing files.
        force: Overwrite existing files.
        request_interval: Seconds to sleep between API calls (free-tier rate limit).
    """
    if not dry_run:
        output_dir.mkdir(parents=True, exist_ok=True)

    # Deduplicate by normalized key so "LERNEN" and "lernen" map to one file.
    seen: dict[str, str] = {}  # normalized_key → original word
    for word in words:
        key = unicodedata.normalize("NFC", word.strip().lower())
        if key and key not in seen:
            seen[key] = word

    unique_words = list(seen.values())
    total = len(unique_words)

    if dry_run:
        for i, word in enumerate(unique_words, 1):
            log.info("[%d/%d] dry-run: would synthesize: %s", i, total, word)
        return

    client = TextToSpeechClient()

    synthesized = 0
    skipped = 0

    for i, word in enumerate(unique_words, 1):
        filename = audio_filename(word)
        dest = output_dir / filename

        if not force and dest.exists():
            log.info("[%d/%d] skip (exists): %s → %s", i, total, word, filename)
            skipped += 1
            continue

        log.info("[%d/%d] synthesize: %s → %s", i, total, word, filename)

        resp = client.synthesize_speech(request={
            "input": {"text": word},
            "voice": {"language_code": "de-DE", "name": voice},
            "audio_config": {"audio_encoding": AudioEncoding.OGG_OPUS},
        })
        dest.write_bytes(resp.audio_content)
        synthesized += 1

        if i < total:
            time.sleep(request_interval)

    log.info(
        "done: %d synthesized, %d skipped (already existed), %d total unique words",
        synthesized, skipped, total,
    )


def _read_german_words(csv_path: Path) -> list[str]:
    """Read unique, non-empty German words from Deutch.csv."""
    words: list[str] = []
    with csv_path.open(encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f, delimiter=";")
        for row in reader:
            german = (row.get("Allemand") or "").strip()
            if german:
                words.append(german)
    return words


def main(argv=None) -> None:
    """CLI entry point — accepts argv for testability."""
    parser = argparse.ArgumentParser(
        description="Generate OGG/Opus pronunciation audio via Google Cloud TTS."
    )
    parser.add_argument("csv_path", help="Path to Deutch.csv")
    parser.add_argument("output_dir", help="Directory to write .ogg files into")
    parser.add_argument(
        "--voice", default="de-DE-Neural2-F",
        help="Google Cloud TTS voice name (default: de-DE-Neural2-F)"
    )
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Log what would be done without calling the API"
    )
    parser.add_argument(
        "--force", action="store_true",
        help="Overwrite existing audio files"
    )
    args = parser.parse_args(argv)

    # Credentials check: fail fast with a clear message before reading CSV.
    try:
        TextToSpeechClient()
    except DefaultCredentialsError:
        print(
            "Error: Google Cloud credentials not found.\n"
            "Set GOOGLE_APPLICATION_CREDENTIALS to the path of your service account JSON.\n"
            "Example: export GOOGLE_APPLICATION_CREDENTIALS=credentials/google-service-account.json",
            file=sys.stderr,
        )
        sys.exit(2)

    csv_path = Path(args.csv_path)
    if not csv_path.exists():
        print(f"Error: CSV file not found: {csv_path}", file=sys.stderr)
        sys.exit(2)

    words = _read_german_words(csv_path)
    log.info("read %d German words from %s", len(words), csv_path)

    generate_audio(
        words,
        Path(args.output_dir),
        voice=args.voice,
        dry_run=args.dry_run,
        force=args.force,
    )


if __name__ == "__main__":
    main()
