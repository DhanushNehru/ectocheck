package claims

import (
	"strings"
	"testing"
)

func TestSplitBulletsAndSentences(t *testing.T) {
	answer := `# My project

The tool reads logs. It runs offline.

- It found 14 errors.
- Nothing was uploaded.`
	got := Split(answer)
	want := []string{
		"The tool reads logs.",
		"It runs offline.",
		"It found 14 errors.",
		"Nothing was uploaded.",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d claims %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("claim %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestSplitKeepsVersionsAndDecimals(t *testing.T) {
	got := Split("It needs Go 1.26. Accuracy reached 87.5 percent.")
	if len(got) != 2 {
		t.Fatalf("expected 2 claims, got %v", got)
	}
	if got[0] != "It needs Go 1.26." {
		t.Errorf("version split wrong: %q", got[0])
	}
}

func TestSplitDropsBoilerplate(t *testing.T) {
	answer := "*This is a submission for the [Challenge](https://dev.to/challenges/x)*\n\n# Title\n\nhttps://example.com\n\nA real claim."
	got := Split(answer)
	if len(got) != 1 || got[0] != "A real claim." {
		t.Fatalf("boilerplate not filtered: %v", got)
	}
}

func TestSentencesJoinsWrappedLines(t *testing.T) {
	got := Split("A claim that wraps\nacross two lines.")
	if len(got) != 1 || !strings.Contains(got[0], "wraps across two lines.") {
		t.Fatalf("wrapped sentence mishandled: %v", got)
	}
}
