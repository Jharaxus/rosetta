package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// fixtureCase mirrors one entry in audio_filename_cases.json.
type fixtureCase struct {
	Note     string `json:"note"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

// loadFixtureCases reads the committed cross-language fixture.
// The fixture lives at resources/fixtures/audio_filename_cases.json relative to the repo root.
func loadFixtureCases(t *testing.T) []fixtureCase {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile = backend/internal/seed/seed_audio_test.go
	// repo root = 3 levels up (seed → internal → backend → rosetta)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	fixturePath := filepath.Join(repoRoot, "resources", "fixtures", "audio_filename_cases.json")

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("cannot read audio_filename_cases.json: %v", err)
	}
	var cases []fixtureCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("cannot parse audio_filename_cases.json: %v", err)
	}
	return cases
}

// TestAudioFilename verifies the cross-language contract:
// Go's audioFilename() must produce identical output to Python's audio_filename().
// Expected values are precomputed and committed in resources/fixtures/audio_filename_cases.json.
func TestAudioFilename(t *testing.T) {
	cases := loadFixtureCases(t)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Note, func(t *testing.T) {
			got := audioFilename(tc.Input)
			if got != tc.Expected {
				t.Errorf("audioFilename(%q)\n  got  %s\n  want %s", tc.Input, got, tc.Expected)
			}
		})
	}
}

// TestAudioFilename_NFDEquivalence explicitly verifies NFC normalisation:
// the same logical string in NFD encoding must produce the same hash as NFC.
func TestAudioFilename_NFDEquivalence(t *testing.T) {
	inputs := []string{"üben", "über", "schöne", "Züge"}
	for _, word := range inputs {
		nfc := norm.NFC.String(word)
		nfd := norm.NFD.String(word)
		if !utf8.ValidString(nfc) || !utf8.ValidString(nfd) {
			t.Skipf("skipping %q: invalid UTF-8 in test environment", word)
		}
		gotNFC := audioFilename(nfc)
		gotNFD := audioFilename(nfd)
		if gotNFC != gotNFD {
			t.Errorf("NFC/NFD mismatch for %q:\n  NFC→%s\n  NFD→%s", word, gotNFC, gotNFD)
		}
	}
}

// TestAudioFilename_Properties verifies structural properties of the output.
func TestAudioFilename_Properties(t *testing.T) {
	cases := []string{"lernen", "sein", "üben", "  LERNEN  "}
	for _, input := range cases {
		got := audioFilename(input)
		if !filepath.IsLocal(got) {
			t.Errorf("audioFilename(%q) = %q is not a local path", input, got)
		}
		if len(got) != 64+4 { // 64-char hex + ".ogg"
			t.Errorf("audioFilename(%q) length = %d, want 68", input, len(got))
		}
		if got[len(got)-4:] != ".ogg" {
			t.Errorf("audioFilename(%q) does not end in .ogg: %q", input, got)
		}
	}
}

// TestAudioFilename_CaseAndWhitespace verifies normalization.
func TestAudioFilename_CaseAndWhitespace(t *testing.T) {
	base := audioFilename("lernen")
	variants := []string{"LERNEN", "Lernen", "  lernen  ", "  LERNEN  ", "\tlernen\n"}
	for _, v := range variants {
		if got := audioFilename(v); got != base {
			t.Errorf("audioFilename(%q) = %q, want same as audioFilename(\"lernen\") = %q", v, got, base)
		}
	}
}
