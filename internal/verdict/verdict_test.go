package verdict

import (
	"errors"
	"strings"
	"testing"

	"github.com/DhanushNehru/ectocheck/internal/retrieve"
	"github.com/DhanushNehru/ectocheck/internal/source"
)

type fakeChatter struct {
	reply string
	err   error
	saw   string
}

func (f *fakeChatter) Chat(system, user string) (string, error) {
	f.saw = user
	return f.reply, f.err
}

func cands() []retrieve.Candidate {
	return []retrieve.Candidate{
		{Chunk: source.Chunk{ID: "E1", File: "log.txt", Line: 1, Text: "backup ran at midnight"}, Score: 4},
		{Chunk: source.Chunk{ID: "E2", File: "log.txt", Line: 5, Text: "upload used zstd"}, Score: 2},
	}
}

func TestSupportedWithValidCitation(t *testing.T) {
	f := &fakeChatter{reply: "VERDICT: SUPPORTED\nEVIDENCE: [1]\nREASON: the log says so"}
	r := Judge(f, "The backup ran at midnight.", cands())
	if r.Status != Supported {
		t.Fatalf("status = %s, want SUPPORTED", r.Status)
	}
	if len(r.Cited) != 1 || r.Cited[0].ID != "E1" {
		t.Errorf("cited = %+v", r.Cited)
	}
	if r.Reason != "the log says so" {
		t.Errorf("reason = %q", r.Reason)
	}
	if !strings.Contains(f.saw, "[1]") {
		t.Error("judge was not shown the evidence")
	}
}

func TestInventedCitationNotTrusted(t *testing.T) {
	f := &fakeChatter{reply: "VERDICT: SUPPORTED\nEVIDENCE: [9]\nREASON: made up"}
	r := Judge(f, "Anything.", cands())
	if r.Status != Unverified {
		t.Fatalf("status = %s, want UNVERIFIED on invented citation", r.Status)
	}
	if !r.BadCitation {
		t.Error("BadCitation not flagged")
	}
	if len(r.Cited) != 0 {
		t.Errorf("invented citation kept: %+v", r.Cited)
	}
}

func TestSupportedWithoutCitationDowngraded(t *testing.T) {
	f := &fakeChatter{reply: "VERDICT: SUPPORTED\nEVIDENCE:\nREASON: trust me"}
	r := Judge(f, "Anything.", cands())
	if r.Status != Unverified {
		t.Fatalf("status = %s, want UNVERIFIED", r.Status)
	}
}

func TestGhostVerdict(t *testing.T) {
	f := &fakeChatter{reply: "VERDICT: GHOST\nREASON: nothing about cats here"}
	r := Judge(f, "The server has a cat.", cands())
	if r.Status != Ghost {
		t.Fatalf("status = %s, want GHOST", r.Status)
	}
}

func TestNoCandidatesMeansGhost(t *testing.T) {
	f := &fakeChatter{reply: "VERDICT: SUPPORTED"}
	r := Judge(f, "Anything.", nil)
	if r.Status != Ghost {
		t.Fatalf("status = %s, want GHOST when no evidence exists", r.Status)
	}
}

func TestModelFailureUnverified(t *testing.T) {
	f := &fakeChatter{err: errors.New("connection refused")}
	r := Judge(f, "Anything.", cands())
	if r.Status != Unverified || !r.ModelError {
		t.Fatalf("status = %s modelError=%v", r.Status, r.ModelError)
	}
}

func TestUnparseableReplyUnverified(t *testing.T) {
	f := &fakeChatter{reply: "hmm, let me think about this..."}
	r := Judge(f, "Anything.", cands())
	if r.Status != Unverified || !r.ModelError {
		t.Fatalf("status = %s", r.Status)
	}
}
