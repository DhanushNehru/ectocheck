package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DhanushNehru/ectocheck/internal/retrieve"
	"github.com/DhanushNehru/ectocheck/internal/source"
	"github.com/DhanushNehru/ectocheck/internal/verdict"
)

func sample() []verdict.Result {
	return []verdict.Result{
		{Claim: "true thing", Status: verdict.Supported, Cited: []retrieve.Candidate{
			{Chunk: source.Chunk{ID: "E1", File: "a.txt", Line: 2, Text: "the true thing evidence"}},
		}},
		{Claim: "invented thing", Status: verdict.Ghost},
		{Claim: "half thing", Status: verdict.Partial},
		{Claim: "unknown thing", Status: verdict.Unverified},
	}
}

func TestSummarize(t *testing.T) {
	s := Summarize(sample())
	if s.Claims != 4 || s.Supported != 1 || s.Ghosts != 1 || s.Partial != 1 || s.Unverified != 1 {
		t.Errorf("bad summary %+v", s)
	}
}

func TestTextReportMentionsGhostsAndCitations(t *testing.T) {
	var buf bytes.Buffer
	s := Summarize(sample())
	WriteText(&buf, sample(), s)
	out := buf.String()
	if !strings.Contains(out, "The answer is haunted.") {
		t.Error("missing haunted verdict")
	}
	if !strings.Contains(out, "[E1] a.txt:2") {
		t.Error("missing citation line")
	}
	if !strings.Contains(out, "4 claims: 1 supported, 1 partial, 1 ghosts, 1 unverified") {
		t.Errorf("bad summary line:\n%s", out)
	}
}

func TestJSONReport(t *testing.T) {
	var buf bytes.Buffer
	s := Summarize(sample())
	if err := WriteJSON(&buf, sample(), s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"ghosts": 1`) || !strings.Contains(out, `"claim": "invented thing"`) {
		t.Errorf("bad json:\n%s", out)
	}
}
