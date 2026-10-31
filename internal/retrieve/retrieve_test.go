package retrieve

import (
	"testing"

	"github.com/DhanushNehru/ectocheck/internal/source"
)

func ev(id, text string) source.Chunk {
	return source.Chunk{ID: id, File: "f", Line: 1, Text: text}
}

func TestTopPicksRelevantChunk(t *testing.T) {
	evidence := []source.Chunk{
		ev("E1", "the backup job runs at midnight and writes to /var/backups"),
		ev("E2", "unrelated notes about dinner plans and shopping"),
		ev("E3", "backups are compressed with zstd before upload"),
	}
	got := Top("a report about backups", "When does the backup job run?", evidence, 2)
	if len(got) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(got))
	}
	if got[0].ID != "E1" {
		t.Errorf("top candidate = %s, want E1 (score %d)", got[0].ID, got[0].Score)
	}
}

func TestTokensDropStopwordsKeepNumbers(t *testing.T) {
	toks := Tokens("The server is on port 22 and it was fine")
	for _, w := range toks {
		if w == "the" || w == "is" || w == "it" || w == "was" {
			t.Errorf("stopword kept: %q", w)
		}
	}
	found := false
	for _, w := range toks {
		if w == "22" {
			found = true
		}
	}
	if !found {
		t.Error("number token dropped")
	}
}

func TestTopWithEmptyEvidence(t *testing.T) {
	if got := Top("ctx", "claim", nil, 3); len(got) != 0 {
		t.Errorf("want no candidates, got %d", len(got))
	}
}
