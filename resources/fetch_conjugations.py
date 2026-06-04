"""
German verb conjugation fetcher — de.wiktionary.org Flexion: pages.

Subcommands
-----------
filter   <input_csv>  <output_csv>   Filter verb rows from the full lexicon.
conjugate <verbs_csv> <output_csv>   Fetch conjugations for each verb.

Output CSV (conjugate): verb;tense;person;conjugation
  verb        — original german column value from the input CSV
  tense       — DB enum string (e.g. praesens_indikativ)
  person      — integer 1–6 (ich/du/er-sie-es/wir/ihr/sie)
  conjugation — comma-separated alternate forms (e.g. seiest, seist)
"""

from __future__ import annotations

import csv
import re
import sys
import time
import logging
from html import unescape
from pathlib import Path
from typing import Iterator

import requests
from bs4 import BeautifulSoup, NavigableString, Tag

logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
log = logging.getLogger(__name__)

# ── Constants ─────────────────────────────────────────────────────────────────

WIKTIONARY_API = "https://de.wiktionary.org/w/api.php"
USER_AGENT = "Rosetta/1.0 (language learning app; contact: hugo.majerczyk@proton.me)"
MAX_RETRIES = 5

# Tense section headings that appear as single-cell rows inside tables
KNOWN_TENSES = frozenset(
    ["Präsens", "Präteritum", "Perfekt", "Plusquamperfekt", "Futur I", "Futur II"]
)

# (section_heading, mood_column_header) → verb_tense enum value
TENSE_MOOD_MAP: dict[tuple[str, str], str] = {
    ("Präsens",        "Indikativ"):     "praesens_indikativ",
    ("Präsens",        "Konjunktiv I"):  "praesens_konjunktiv_1",
    ("Präteritum",     "Indikativ"):     "praeteritum_indikativ",
    ("Präteritum",     "Konjunktiv II"): "praeteritum_konjunktiv_2",
    ("Perfekt",        "Indikativ"):     "perfekt_indikativ",
    ("Perfekt",        "Konjunktiv I"):  "perfekt_konjunktiv_1",
    ("Plusquamperfekt","Indikativ"):     "plusquamperfekt_indikativ",
    ("Plusquamperfekt","Konjunktiv II"): "plusquamperfekt_konjunktiv_2",
    ("Futur I",        "Indikativ"):     "futur_1_indikativ",
    ("Futur I",        "Konjunktiv I"):  "futur_1_konjunktiv_1",
    ("Futur I",        "Konjunktiv II"): "futur_1_konjunktiv_2",
    ("Futur II",       "Indikativ"):     "futur_2_indikativ",
    ("Futur II",       "Konjunktiv I"):  "futur_2_konjunktiv_1",
    ("Futur II",       "Konjunktiv II"): "futur_2_konjunktiv_2",
}

# Row-header text → person integer
PERSON_MAP: dict[str, int] = {
    "1. Person Singular": 1,
    "2. Person Singular": 2,
    "3. Person Singular": 3,
    "1. Person Plural":   4,
    "2. Person Plural":   5,
    "3. Person Plural":   6,
}

# Row-header text → pronoun prefix to strip from form text
PRONOUN_PREFIX: dict[str, str] = {
    "1. Person Singular": "ich ",
    "2. Person Singular": "du ",
    "3. Person Singular": "er/sie/es ",
    "1. Person Plural":   "wir ",
    "2. Person Plural":   "ihr ",
    "3. Person Plural":   "sie ",
}

# Imperativ person rows → person integer
IMPERATIV_PERSON_MAP: dict[str, int] = {
    "2. Person Singular": 2,
    "2. Person Plural":   5,
    "Höflichkeitsform":   6,
}


# ── HTML cell helpers ─────────────────────────────────────────────────────────

def _collect_cell_text(node: Tag) -> list[str]:
    """Recursively collect text segments, inserting newlines at <br> tags."""
    parts: list[str] = []
    for child in node.children:
        if isinstance(child, NavigableString):
            parts.append(str(child))
        elif isinstance(child, Tag):
            if child.name == "br":
                parts.append("\n")
            elif child.name in ("sup", "ref"):
                pass  # skip footnote superscripts
            else:
                parts.extend(_collect_cell_text(child))
    return parts


def cell_forms(cell: Tag, strip_pronoun: str = "") -> list[str]:
    """
    Extract a list of conjugated forms from a table cell.

    Handles:
    - <br/> separated alternate forms (e.g. arbeite! / arbeit!)
    - Comma-and-<br/> separated alternates (e.g. seiest,<br/>seist)
    - Pronoun prefix stripping (e.g. "du bist" → "bist")
    - Trailing "!" stripping for imperativ
    - "ungebräuchlich:" prefix stripping
    - Em-dash (—) cells → returns empty list
    """
    raw = "".join(_collect_cell_text(cell))
    forms: list[str] = []
    for line in raw.split("\n"):
        form = line.strip().rstrip(",").strip()
        if not form or form in ("—", "-", "–"):
            continue
        if form.startswith("ungebräuchlich:"):
            form = form[len("ungebräuchlich:"):].strip()
        form = form.rstrip("!").strip()
        if strip_pronoun and form.lower().startswith(strip_pronoun.lower()):
            form = form[len(strip_pronoun):].strip()
        if form:
            forms.append(form)
    return forms


def _normalize_heading(text: str) -> str:
    """Strip footnote markers and normalize whitespace in heading text."""
    text = re.sub(r"\[\d+\]", "", text)
    text = text.replace(" ", " ")  # non-breaking space
    return text.strip()


# ── Core parser ───────────────────────────────────────────────────────────────

def parse_conjugations(html: str) -> list[dict]:
    """
    Parse a Flexion: page HTML into a list of conjugation records.

    Returns a list of dicts:
        {"tense": str, "person": int, "forms": list[str]}

    Only Aktiv voice forms are extracted. Vorgangspassiv and Zustandspassiv
    are ignored.
    """
    soup = BeautifulSoup(html, "lxml")
    tables = soup.find_all("table")
    if len(tables) < 2:
        return []

    results: list[dict] = []

    # Table 0 is always the infinitive/participle table — skip.
    # Table 1 is always the Imperativ table.
    # Tables 2+ contain the finite conjugations (Präsens, Präteritum, etc.)

    if len(tables) >= 2:
        results.extend(_parse_imperativ_table(tables[1]))

    for table in tables[2:]:
        results.extend(_parse_conjugation_table(table))

    return results


def _parse_imperativ_table(table: Tag) -> list[dict]:
    """
    Parse the Imperativ table (always Table 1 on Flexion: pages).

    Takes only the 'Präsens Aktiv' column (index 1 in the header row).
    """
    rows = table.find_all("tr")
    results: list[dict] = []

    # rows[0]: "Imperative" header — skip
    # rows[1]: mood column headers
    # rows[2+]: person rows

    if len(rows) < 3:
        return results

    # Verify 'Präsens Aktiv' is at column 1
    header_cells = rows[1].find_all(["th", "td"])
    if not header_cells or len(header_cells) < 2:
        return results

    for row in rows[2:]:
        cells = row.find_all(["th", "td"])
        if not cells:
            continue
        person_text = _normalize_heading(cells[0].get_text(strip=True))
        person_num = IMPERATIV_PERSON_MAP.get(person_text)
        if person_num is None:
            continue
        if len(cells) < 2:
            continue
        forms = cell_forms(cells[1])  # 'Präsens Aktiv' column, no pronoun to strip
        if forms:
            results.append({"tense": "imperativ", "person": person_num, "forms": forms})

    return results


def _parse_conjugation_table(table: Tag) -> list[dict]:
    """
    Parse a conjugation table that may contain multiple tense sections.

    Each section has the structure:
        [tense_name]                  (single-cell row, e.g. 'Präsens')
        [voice headers with colspan]  (Person / Aktiv / Vorgangspassiv / ...)
        [mood sub-headers]            ('' or 'Person' / Indikativ / Konjunktiv I ...)
        [person data rows × 6]        (1.Person Singular → forms)
        [Text]                        (separator before next section)

    Only the Aktiv voice columns are extracted. aktiv_colspan (2 or 3) is read
    from the colspan attribute on the 'Aktiv' <th>/<td> cell.
    """
    # State machine states
    SCAN = "SCAN"
    EXPECT_VOICE = "EXPECT_VOICE"
    EXPECT_MOOD = "EXPECT_MOOD"
    READ_PERSONS = "READ_PERSONS"

    state = SCAN
    current_tense: str | None = None
    aktiv_colspan = 0
    aktiv_moods: list[str] = []  # mood names for Aktiv columns (length = aktiv_colspan)
    results: list[dict] = []

    for row in table.find_all("tr"):
        cells = row.find_all(["th", "td"])
        if not cells:
            continue

        row_text_0 = _normalize_heading(cells[0].get_text(strip=True))

        # ── Single-cell rows ──────────────────────────────────────────────────
        if len(cells) == 1:
            if row_text_0 in KNOWN_TENSES:
                current_tense = row_text_0
                state = EXPECT_VOICE
            else:
                # "Text" separator or unknown single-cell row
                state = SCAN
            continue

        # ── Voice header row: contains a cell with text "Aktiv" ──────────────
        if state in (SCAN, EXPECT_VOICE):
            for cell in cells:
                if cell.get_text(strip=True) == "Aktiv":
                    aktiv_colspan = int(cell.get("colspan", 1))
                    state = EXPECT_MOOD
                    break
            else:
                # No "Aktiv" cell — might be the mood header row if we missed
                # the voice header (shouldn't happen with well-formed pages)
                pass
            continue

        # ── Mood header row: immediately follows the voice header row ─────────
        if state == EXPECT_MOOD:
            # cells[0] is either '' or 'Person'; cells[1..aktiv_colspan] are mood names
            aktiv_moods = [
                cells[i].get_text(strip=True)
                for i in range(1, min(aktiv_colspan + 1, len(cells)))
            ]
            state = READ_PERSONS
            continue

        # ── Person data rows ──────────────────────────────────────────────────
        if state == READ_PERSONS:
            if row_text_0 not in PERSON_MAP:
                # End of person rows (separator, next tense name, etc.)
                # Re-process this row as a potential tense name
                if row_text_0 in KNOWN_TENSES:
                    current_tense = row_text_0
                    state = EXPECT_VOICE
                else:
                    state = SCAN
                continue

            person_num = PERSON_MAP[row_text_0]
            pronoun = PRONOUN_PREFIX.get(row_text_0, "")

            for i, mood in enumerate(aktiv_moods):
                col_idx = i + 1  # data column index (1-based, 0 is person label)
                if col_idx >= len(cells):
                    continue

                tense_key = (current_tense, mood)
                tense_enum = TENSE_MOOD_MAP.get(tense_key)
                if not tense_enum:
                    continue

                forms = cell_forms(cells[col_idx], strip_pronoun=pronoun)
                if forms:
                    results.append({
                        "tense": tense_enum,
                        "person": person_num,
                        "forms": forms,
                    })

    return results


# ── Wiktionary HTTP client ────────────────────────────────────────────────────

def fetch_wiktionary(verb: str) -> str | None:
    """
    Fetch the rendered HTML of a Flexion:<verb> page from de.wiktionary.org.

    Returns the HTML string, or None if the page doesn't exist or all retries
    are exhausted.

    Rate-limiting: sleeps 1 second before every request. Uses exponential
    backoff on maxlag and 5xx responses.
    """
    params = {
        "action": "parse",
        "page": f"Flexion:{verb}",
        "prop": "text",
        "format": "json",
        "formatversion": "2",
        "maxlag": "5",
    }
    headers = {"User-Agent": USER_AGENT}

    for attempt in range(MAX_RETRIES):
        time.sleep(1)
        try:
            resp = requests.get(WIKTIONARY_API, params=params, headers=headers, timeout=30)
        except requests.RequestException as exc:
            wait = 2 ** attempt
            log.warning("network error for %s (attempt %d/%d), retrying in %ds: %s",
                        verb, attempt + 1, MAX_RETRIES, wait, exc)
            time.sleep(wait)
            continue

        if resp.status_code == 200:
            data = resp.json()
            if "error" in data:
                code = data["error"].get("code", "")
                if code == "missingtitle":
                    log.warning("Flexion:%s not found on Wiktionary", verb)
                    return None
                if code == "maxlag":
                    retry_after = int(resp.headers.get("retry-after", 2 ** attempt))
                    log.warning("maxlag for %s, waiting %ds", verb, retry_after)
                    time.sleep(retry_after)
                    continue
                log.warning("API error for %s: %s", verb, data["error"])
                return None
            return data["parse"]["text"]

        if resp.status_code >= 500:
            wait = 2 ** attempt
            log.warning("HTTP %d for %s (attempt %d/%d), retrying in %ds",
                        resp.status_code, verb, attempt + 1, MAX_RETRIES, wait)
            time.sleep(wait)
            continue

        log.warning("unexpected HTTP %d for %s", resp.status_code, verb)
        return None

    log.error("max retries exceeded for %s, skipping", verb)
    return None


# ── Infinitive extraction ─────────────────────────────────────────────────────

def extract_wiktionary_infinitive(german_csv_value: str) -> str | None:
    """
    Convert a raw CSV 'german' column value into a Wiktionary-queryable infinitive.

    Returns None if a clean single-word infinitive cannot be derived.
    """
    s = german_csv_value.strip().strip('"\'')
    # Take first alternative (split on ' / ' or ' ; ')
    s = re.split(r"\s+[/;]\s+", s)[0].strip()
    # Remove separable-verb slash: auf/räumen → aufräumen
    s = s.replace("/", "")
    # Strip reflexive prefix
    if s.startswith("sich "):
        s = s[5:].strip()
    # Strip parenthetical notes
    s = re.sub(r"\(.*?\)", "", s).strip()
    # Strip trailing punctuation and whitespace
    s = s.strip(".,;:!?").strip()
    # Reject if still contains a space (not a single word)
    if not s or " " in s:
        return None
    return s


# ── CLI commands ──────────────────────────────────────────────────────────────

def cmd_filter(input_csv: str, output_csv: str) -> None:
    """Filter verb rows from the full lexicon CSV."""
    count = 0
    with (
        open(input_csv, newline="", encoding="utf-8") as fin,
        open(output_csv, "w", newline="", encoding="utf-8") as fout,
    ):
        reader = csv.reader(fin, delimiter=";")
        writer = csv.writer(fout, delimiter=";")
        header = next(reader)
        writer.writerow(["french", "german", "assimil_lesson", "category", "regularity"])
        for row in reader:
            if len(row) >= 4 and row[3].strip() == "Verb":
                writer.writerow(row)
                count += 1

    log.info("filter: wrote %d verb rows to %s", count, output_csv)


def cmd_conjugate(verbs_csv: str, output_csv: str) -> None:
    """Fetch conjugations from Wiktionary for each verb in the input CSV."""
    with open(verbs_csv, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f, delimiter=";")
        rows = list(reader)

    # Deduplicate by german value to avoid redundant API calls
    seen_german: set[str] = set()
    unique_verbs: list[str] = []
    for row in rows:
        german = row.get("german", "").strip()
        if german not in seen_german:
            seen_german.add(german)
            unique_verbs.append(german)

    total = len(unique_verbs)
    log.info("conjugate: processing %d unique verbs", total)

    with open(output_csv, "w", newline="", encoding="utf-8") as fout:
        writer = csv.writer(fout, delimiter=";")
        writer.writerow(["verb", "tense", "person", "conjugation"])

        for i, german in enumerate(unique_verbs, 1):
            infinitive = extract_wiktionary_infinitive(german)
            if infinitive is None:
                log.warning("[%d/%d] %s → SKIPPED (cannot derive infinitive)",
                            i, total, german)
                continue

            html = fetch_wiktionary(infinitive)
            if html is None:
                log.warning("[%d/%d] %s → SKIPPED (no Flexion page)", i, total, infinitive)
                continue

            conjugations = parse_conjugations(html)
            if not conjugations:
                log.warning("[%d/%d] %s → SKIPPED (no conjugations parsed)", i, total, infinitive)
                continue

            for conj in conjugations:
                writer.writerow([
                    german,
                    conj["tense"],
                    conj["person"],
                    ", ".join(conj["forms"]),
                ])

            log.info("[%d/%d] %s → %d forms", i, total, infinitive, len(conjugations))


# ── Entry point ───────────────────────────────────────────────────────────────

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage:")
        print("  python3 fetch_conjugations.py filter <input_csv> <output_csv>")
        print("  python3 fetch_conjugations.py conjugate <verbs_csv> <output_csv>")
        sys.exit(1)

    command = sys.argv[1]

    if command == "filter":
        if len(sys.argv) != 4:
            print("Usage: python3 fetch_conjugations.py filter <input_csv> <output_csv>")
            sys.exit(1)
        cmd_filter(sys.argv[2], sys.argv[3])

    elif command == "conjugate":
        if len(sys.argv) != 4:
            print("Usage: python3 fetch_conjugations.py conjugate <verbs_csv> <output_csv>")
            sys.exit(1)
        cmd_conjugate(sys.argv[2], sys.argv[3])

    else:
        print(f"Unknown command: {command!r}")
        print("Use: filter | conjugate")
        sys.exit(1)
