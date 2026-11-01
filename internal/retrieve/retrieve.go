// Package retrieve finds the evidence chunks most likely to be relevant to
// a claim, using deterministic token-overlap scoring. Retrieval is
// deliberately simple and fully inspectable: the same claim always yields
// the same candidates, and the score is just shared-word counts with
// stopwords removed.
package retrieve

import (
	"sort"
	"strings"
	"unicode"

	"github.com/DhanushNehru/ectocheck/internal/source"
)

// Candidate is an evidence chunk plus its retrieval score.
type Candidate struct {
	source.Chunk
	Score int
}

var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"of": true, "to": true, "in": true, "on": true, "at": true, "for": true,
	"with": true, "by": true, "from": true, "as": true, "it": true, "its": true,
	"this": true, "that": true, "these": true, "those": true, "i": true, "we": true,
	"you": true, "he": true, "she": true, "they": true, "my": true, "our": true,
	"your": true, "their": true, "his": true, "her": true, "not": true, "no": true,
	"so": true, "if": true, "then": true, "than": true, "too": true, "very": true,
	"can": true, "will": true, "would": true, "could": true, "should": true,
	"do": true, "does": true, "did": true, "has": true, "have": true, "had": true,
	"also": true, "just": true, "about": true, "into": true, "over": true,
}

// Tokens extracts lowercase word tokens, dropping stopwords and one-letter
// words. Numbers are kept: they are often the load-bearing part of a claim.
func Tokens(s string) []string {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 2 && !isNumber(w) {
			continue
		}
		if stopwords[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

func isNumber(w string) bool {
	for _, r := range w {
		if !unicode.IsNumber(r) {
			return false
		}
	}
	return len(w) > 0
}

// Top returns up to k evidence chunks with the highest token overlap with
// the claim. Context (the whole answer) contributes half weight, so a claim
// like "It also retries twice" still retrieves chunks about the right topic.
// Chunks scoring zero are returned last, so the judge always sees something,
// and the report can note when retrieval found nothing real.
func Top(context, claim string, evidence []source.Chunk, k int) []Candidate {
	claimToks := tokenSet(Tokens(claim))
	ctxToks := tokenSet(Tokens(context))

	scored := make([]Candidate, 0, len(evidence))
	for _, ev := range evidence {
		evToks := tokenSet(Tokens(ev.Text))
		score := 0
		for t := range claimToks {
			if evToks[t] {
				score += 2
			}
		}
		for t := range ctxToks {
			if evToks[t] {
				score++
			}
		}
		scored = append(scored, Candidate{Chunk: ev, Score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if k > len(scored) {
		k = len(scored)
	}
	if k < 0 {
		k = 0
	}
	return scored[:k]
}

func tokenSet(toks []string) map[string]bool {
	m := make(map[string]bool, len(toks))
	for _, t := range toks {
		m[t] = true
	}
	return m
}
