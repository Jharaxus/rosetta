"""
Tests for fetch_conjugations.py.

Test structure
--------------
1. TestEndToEnd        — full conjugate pipeline for sein/arbeiten/verstehen using
                         fixture JSON as mocked HTTP; compared to expected_conjugations.csv
2. TestParseConjugationsSein      — HTML parsing unit tests for 'sein'
3. TestParseConjugationsArbeiten  — HTML parsing unit tests for 'arbeiten'
4. TestParseConjugationsVerstehen — HTML parsing unit tests for 'verstehen'
5. TestAlternateForms             — cells with multiple forms
6. TestFetchWiktionary            — HTTP client error handling

TDD note: fixture JSON files and expected_conjugations.csv are committed.
They must not be auto-regenerated.
"""

from __future__ import annotations

import csv
import json
import time
from pathlib import Path
from typing import Any

import pytest
import responses as rsps_lib

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

FIXTURES = Path(__file__).parent.parent / "fixtures"

WIKTIONARY_API = "https://de.wiktionary.org/w/api.php"


def load_html(name: str) -> str:
    """Load rendered HTML from a stored Wiktionary API JSON fixture."""
    data = json.loads((FIXTURES / f"flexion_{name}.json").read_text())
    return data["parse"]["text"]


def load_html_bytes(name: str) -> bytes:
    return (FIXTURES / f"flexion_{name}.json").read_bytes()


def load_expected_csv() -> set[tuple[str, str, int, str]]:
    """Return expected (verb, tense, person, conjugation) as an unordered set."""
    rows: set[tuple[str, str, int, str]] = set()
    with open(FIXTURES / "expected_conjugations.csv", newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f, delimiter=";")
        for row in reader:
            rows.add((row["verb"], row["tense"], int(row["person"]), row["conjugation"]))
    return rows


def make_tense_map(name: str) -> dict[tuple[str, int], list[str]]:
    """Parse fixture HTML and return {(tense_enum, person_int): [forms]} dict."""
    from fetch_conjugations import parse_conjugations
    return {
        (r["tense"], r["person"]): r["forms"]
        for r in parse_conjugations(load_html(name))
    }


# ---------------------------------------------------------------------------
# 1. End-to-end test
# ---------------------------------------------------------------------------

class TestEndToEnd:
    """
    Run the full conjugate pipeline against mocked HTTP responses and verify
    that the output CSV matches expected_conjugations.csv (order-independent).
    """

    @rsps_lib.activate
    def test_output_matches_expected_csv(self, tmp_path):
        from fetch_conjugations import cmd_conjugate

        # Build a small verbs CSV with sein, arbeiten, verstehen
        verbs_csv = tmp_path / "verbs.csv"
        verbs_csv.write_text(
            "french;german;assimil_lesson;category;regularity\n"
            "être;sein;1;Verb;irregular\n"
            "travailler;arbeiten;1;Verb;regular\n"
            "comprendre;verstehen;1;Verb;irregular\n",
            encoding="utf-8",
        )

        # Mock the three Wiktionary API calls
        for verb_name in ("sein", "arbeiten", "verstehen"):
            rsps_lib.add(
                rsps_lib.GET,
                WIKTIONARY_API,
                body=load_html_bytes(verb_name),
                status=200,
                content_type="application/json",
                match_querystring=False,
            )

        output_csv = tmp_path / "out.csv"

        # Patch time.sleep to avoid slowdowns during tests
        import unittest.mock as mock
        with mock.patch("time.sleep"):
            cmd_conjugate(str(verbs_csv), str(output_csv))

        # Load actual output as a set of tuples
        actual: set[tuple[str, str, int, str]] = set()
        with open(output_csv, newline="", encoding="utf-8") as f:
            for row in csv.DictReader(f, delimiter=";"):
                actual.add((row["verb"], row["tense"], int(row["person"]), row["conjugation"]))

        expected = load_expected_csv()

        missing = expected - actual
        extra = actual - expected
        assert not missing, f"Missing rows in output: {sorted(missing)[:5]}"
        assert not extra, f"Unexpected extra rows in output: {sorted(extra)[:5]}"


# ---------------------------------------------------------------------------
# 2. Unit tests — HTML parsing for 'sein'
# ---------------------------------------------------------------------------

class TestParseConjugationsSein:
    """Unit tests for parse_conjugations() using the fixture HTML for 'sein'."""

    def setup_method(self):
        self.tm = make_tense_map("sein")

    def test_praesens_indikativ_person_1_is_bin(self):
        assert "bin" in self.tm[("praesens_indikativ", 1)]

    def test_praesens_indikativ_person_2_is_bist(self):
        assert "bist" in self.tm[("praesens_indikativ", 2)]

    def test_praesens_indikativ_person_3_is_ist(self):
        assert "ist" in self.tm[("praesens_indikativ", 3)]

    def test_praesens_indikativ_person_4_is_sind(self):
        assert "sind" in self.tm[("praesens_indikativ", 4)]

    def test_praesens_indikativ_person_5_is_seid(self):
        assert "seid" in self.tm[("praesens_indikativ", 5)]

    def test_praesens_indikativ_person_6_is_sind(self):
        assert "sind" in self.tm[("praesens_indikativ", 6)]

    def test_praesens_indikativ_all_six_persons_present(self):
        persons = {person for (tense, person) in self.tm if tense == "praesens_indikativ"}
        assert persons == {1, 2, 3, 4, 5, 6}

    def test_praeteritum_indikativ_person_1_is_war(self):
        assert "war" in self.tm[("praeteritum_indikativ", 1)]

    def test_praeteritum_indikativ_person_2_is_warst(self):
        assert "warst" in self.tm[("praeteritum_indikativ", 2)]

    def test_praeteritum_indikativ_person_4_is_waren(self):
        assert "waren" in self.tm[("praeteritum_indikativ", 4)]

    def test_praeteritum_indikativ_person_5_is_wart(self):
        assert "wart" in self.tm[("praeteritum_indikativ", 5)]

    def test_perfekt_indikativ_person_1_is_bin_gewesen(self):
        assert "bin gewesen" in self.tm[("perfekt_indikativ", 1)]

    def test_futur_1_indikativ_person_1_is_werde_sein(self):
        assert "werde sein" in self.tm[("futur_1_indikativ", 1)]

    def test_imperativ_person_2_is_sei(self):
        assert "sei" in self.tm[("imperativ", 2)]

    def test_imperativ_person_5_is_seid(self):
        assert "seid" in self.tm[("imperativ", 5)]

    def test_imperativ_person_6_is_seien_sie(self):
        assert "seien Sie" in self.tm[("imperativ", 6)]

    def test_tense_enum_strings_are_all_valid(self):
        from fetch_conjugations import TENSE_MOOD_MAP
        valid_enums = set(TENSE_MOOD_MAP.values()) | {"imperativ"}
        for (tense, _) in self.tm:
            assert tense in valid_enums, f"Unknown tense enum: {tense!r}"

    def test_no_empty_forms(self):
        for (tense, person), forms in self.tm.items():
            for f in forms:
                assert f.strip(), (
                    f"Empty form for tense={tense!r} person={person}"
                )

    def test_no_dash_forms(self):
        for (tense, person), forms in self.tm.items():
            for f in forms:
                assert f not in ("—", "-", "–"), (
                    f"Dash stored as form for tense={tense!r} person={person}"
                )


# ---------------------------------------------------------------------------
# 3. Unit tests — HTML parsing for 'arbeiten'
# ---------------------------------------------------------------------------

class TestParseConjugationsArbeiten:
    """Unit tests for parse_conjugations() using the fixture HTML for 'arbeiten'."""

    def setup_method(self):
        self.tm = make_tense_map("arbeiten")

    def test_praesens_indikativ_person_1_is_arbeite(self):
        assert "arbeite" in self.tm[("praesens_indikativ", 1)]

    def test_praesens_indikativ_person_2_is_arbeitest(self):
        assert "arbeitest" in self.tm[("praesens_indikativ", 2)]

    def test_praesens_indikativ_person_3_is_arbeitet(self):
        assert "arbeitet" in self.tm[("praesens_indikativ", 3)]

    def test_praesens_indikativ_person_4_is_arbeiten(self):
        assert "arbeiten" in self.tm[("praesens_indikativ", 4)]

    def test_praesens_indikativ_person_5_is_arbeitet(self):
        assert "arbeitet" in self.tm[("praesens_indikativ", 5)]

    def test_praesens_indikativ_person_6_is_arbeiten(self):
        assert "arbeiten" in self.tm[("praesens_indikativ", 6)]

    def test_praeteritum_indikativ_person_1_is_arbeitete(self):
        assert "arbeitete" in self.tm[("praeteritum_indikativ", 1)]

    def test_perfekt_indikativ_person_1_is_habe_gearbeitet(self):
        assert "habe gearbeitet" in self.tm[("perfekt_indikativ", 1)]

    def test_futur_1_indikativ_person_1_is_werde_arbeiten(self):
        assert "werde arbeiten" in self.tm[("futur_1_indikativ", 1)]

    def test_imperativ_person_2_contains_arbeite(self):
        assert "arbeite" in self.tm[("imperativ", 2)]

    def test_imperativ_person_5_is_arbeitet(self):
        assert "arbeitet" in self.tm[("imperativ", 5)]


# ---------------------------------------------------------------------------
# 4. Unit tests — HTML parsing for 'verstehen'
# ---------------------------------------------------------------------------

class TestParseConjugationsVerstehen:
    """Unit tests for parse_conjugations() using the fixture HTML for 'verstehen'."""

    def setup_method(self):
        self.tm = make_tense_map("verstehen")

    def test_praesens_indikativ_person_1_is_verstehe(self):
        assert "verstehe" in self.tm[("praesens_indikativ", 1)]

    def test_praesens_indikativ_person_2_is_verstehst(self):
        assert "verstehst" in self.tm[("praesens_indikativ", 2)]

    def test_praesens_indikativ_person_3_is_versteht(self):
        assert "versteht" in self.tm[("praesens_indikativ", 3)]

    def test_praeteritum_indikativ_person_1_is_verstand(self):
        assert "verstand" in self.tm[("praeteritum_indikativ", 1)]

    def test_praeteritum_indikativ_person_4_is_verstanden(self):
        assert "verstanden" in self.tm[("praeteritum_indikativ", 4)]

    def test_perfekt_indikativ_person_1_is_habe_verstanden(self):
        assert "habe verstanden" in self.tm[("perfekt_indikativ", 1)]

    def test_imperativ_person_2_contains_versteh(self):
        assert "versteh" in self.tm[("imperativ", 2)]


# ---------------------------------------------------------------------------
# 5. Unit tests — alternate forms handling
# ---------------------------------------------------------------------------

class TestAlternateForms:
    """Tests that cells with multiple forms (<br/>-separated) are handled correctly."""

    def setup_method(self):
        self.tm = make_tense_map("sein")

    def test_sein_konjunktiv_1_praesens_person_2_has_multiple_forms(self):
        """sein Konjunktiv I Präsens p2 must contain both 'seiest' and 'seist'."""
        forms = self.tm.get(("praesens_konjunktiv_1", 2), [])
        assert "seiest" in forms, f"seiest not in {forms}"
        assert "seist" in forms, f"seist not in {forms}"
        assert len(forms) >= 2

    def test_sein_konjunktiv_2_praeteritum_person_2_has_multiple_forms(self):
        """sein Konjunktiv II Präteritum p2 must contain both 'wärest' and 'wärst'."""
        forms = self.tm.get(("praeteritum_konjunktiv_2", 2), [])
        assert "wärest" in forms or "wärst" in forms, f"Expected alternates in {forms}"
        assert len(forms) >= 2

    def test_arbeiten_imperativ_person_2_has_multiple_forms(self):
        """arbeiten Imperativ p2 has both 'arbeite' and 'arbeit'."""
        tm = make_tense_map("arbeiten")
        forms = tm.get(("imperativ", 2), [])
        assert "arbeite" in forms, f"arbeite not in {forms}"
        assert "arbeit" in forms, f"arbeit not in {forms}"
        assert len(forms) >= 2

    def test_verstehen_imperativ_person_2_has_multiple_forms(self):
        """verstehen Imperativ p2 has both 'versteh' and 'verstehe'."""
        tm = make_tense_map("verstehen")
        forms = tm.get(("imperativ", 2), [])
        assert "versteh" in forms, f"versteh not in {forms}"
        assert "verstehe" in forms, f"verstehe not in {forms}"


# ---------------------------------------------------------------------------
# 6. Unit tests — HTTP client (fetch_wiktionary)
# ---------------------------------------------------------------------------

class TestFetchWiktionary:
    """Unit tests for the fetch_wiktionary() HTTP client."""

    @rsps_lib.activate
    def test_returns_none_on_missing_title(self):
        """fetch_wiktionary returns None when API reports missingtitle."""
        import unittest.mock as mock
        from fetch_conjugations import fetch_wiktionary

        rsps_lib.add(
            rsps_lib.GET,
            WIKTIONARY_API,
            json={"error": {"code": "missingtitle", "info": "The page does not exist."}},
            status=200,
        )
        with mock.patch("time.sleep"):
            result = fetch_wiktionary("nichtexistierendesverbxyz")
        assert result is None

    @rsps_lib.activate
    def test_returns_none_after_max_retries_on_5xx(self):
        """fetch_wiktionary returns None (no exception) after repeated 500 responses."""
        import unittest.mock as mock
        from fetch_conjugations import fetch_wiktionary, MAX_RETRIES

        for _ in range(MAX_RETRIES):
            rsps_lib.add(rsps_lib.GET, WIKTIONARY_API, status=500, body="Server Error")

        with mock.patch("time.sleep"):
            result = fetch_wiktionary("irgendeinverb")
        assert result is None

    @rsps_lib.activate
    def test_retries_on_maxlag_then_succeeds(self):
        """fetch_wiktionary retries after a maxlag error and returns HTML on success."""
        import unittest.mock as mock
        from fetch_conjugations import fetch_wiktionary

        # First call: maxlag error
        rsps_lib.add(
            rsps_lib.GET,
            WIKTIONARY_API,
            json={"error": {"code": "maxlag", "info": "Waiting for lag."}},
            status=200,
            headers={"retry-after": "1"},
        )
        # Second call: success
        rsps_lib.add(
            rsps_lib.GET,
            WIKTIONARY_API,
            json={"parse": {"text": "<html>ok</html>"}},
            status=200,
        )

        with mock.patch("time.sleep"):
            result = fetch_wiktionary("lernen")
        assert result == "<html>ok</html>"

    @rsps_lib.activate
    def test_returns_html_on_success(self):
        """fetch_wiktionary returns the HTML string when API responds normally."""
        import unittest.mock as mock
        from fetch_conjugations import fetch_wiktionary

        rsps_lib.add(
            rsps_lib.GET,
            WIKTIONARY_API,
            json={"parse": {"text": "<table>conjugations</table>"}},
            status=200,
        )
        with mock.patch("time.sleep"):
            result = fetch_wiktionary("lernen")
        assert result == "<table>conjugations</table>"


# ---------------------------------------------------------------------------
# 7. Unit tests — extract_wiktionary_infinitive
# ---------------------------------------------------------------------------

class TestExtractWiktionaryInfinitive:
    """Unit tests for the infinitive extraction helper."""

    def test_simple_verb(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        assert extract_wiktionary_infinitive("lernen") == "lernen"

    def test_strips_whitespace(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        assert extract_wiktionary_infinitive(" abnehmen ") == "abnehmen"

    def test_separable_verb_slash_removed(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        assert extract_wiktionary_infinitive("auf/räumen") == "aufräumen"

    def test_takes_first_alternative(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        result = extract_wiktionary_infinitive("sprechen / reden")
        assert result == "sprechen"

    def test_strips_reflexive_sich(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        assert extract_wiktionary_infinitive("sich freuen") == "freuen"

    def test_strips_parenthetical(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        result = extract_wiktionary_infinitive("abhängen (von)")
        assert result == "abhängen"

    def test_returns_none_for_phrase(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        assert extract_wiktionary_infinitive("es gibt") is None

    def test_returns_none_for_empty(self):
        from fetch_conjugations import extract_wiktionary_infinitive
        assert extract_wiktionary_infinitive("") is None
