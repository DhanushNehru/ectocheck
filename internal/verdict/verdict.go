// Package verdict asks a local model to judge one claim against retrieved
// evidence, then checks the model's own work: every evidence ID it cites
// must be one it was actually shown. A verdict that invents citations is
// not trusted — the claim falls back to UNVERIFIED and the report says why.
package verdict

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/DhanushNehru/ectocheck/internal/retrieve"
)

// Status is the verdict for one claim.
type Status string

const (
	Supported  Status = "SUPPORTED"  // evidence backs the whole claim
	Partial    Status = "PARTIAL"    // some of the claim is backed, some is not
	Ghost      Status = "GHOST"      // nothing in the sources backs it
	Unverified Status = "UNVERIFIED" // the check itself could not run cleanly
)

// Result is one judged claim.
type Result struct {
	Claim       string
	Status      Status
	Cited       []retrieve.Candidate // valid citations only, in citation order
	Reason      string               // the model's one-line reason, or the fallback note
	Retrieved   int                  // how many candidates the judge was shown
	TopScore    int                  // retrieval score of the best candidate (0 = nothing relevant found)
	BadCitation bool                 // model cited evidence it was never shown
	ModelError  bool                 // model call failed or reply was unparseable
}

var (
	verdictRe = regexp.MustCompile(`(?i)\b(SUPPORTED|PARTIAL|GHOST)\b`)
	// The model cites evidence positionally ([1], [2], ...) because small
	// models handle small integers far more reliably than global IDs.
	// Positions map back to the real chunk IDs for the report.
	citeRe = regexp.MustCompile(`\[(\d{1,2})\]`)
)

const systemPrompt = `You are an evidence judge. You decide whether one claim is backed by the evidence you are shown, and nothing else.

Reply in exactly this shape, three lines, in this order:
VERDICT: SUPPORTED | PARTIAL | GHOST
EVIDENCE: [1], [3]
REASON: one short sentence

Example, for the claim "The backup ran at midnight" with evidence "[1] backup started 02:00" and "[2] dinner plans":
VERDICT: GHOST
EVIDENCE:
REASON: [1] mentions a backup but at 02:00, not midnight.

Rules:
- The EVIDENCE line is required. Copy the bracketed numbers of the evidence that backs the claim. Never skip this line.
- SUPPORTED means the evidence backs every part of the claim.
- PARTIAL means the evidence backs some of it, but a part is unbacked or off.
- GHOST means none of the evidence backs the claim. Use GHOST when the evidence is merely about the same topic but does not back what the claim actually says. For GHOST, leave EVIDENCE empty.
- Cite only the bracketed numbers of the evidence you were shown.
- Never use knowledge from outside the evidence.`

// Chatter is anything that can answer a system+user prompt. *ollama.Client
// satisfies it; tests substitute a fake.
type Chatter interface {
	Chat(system, user string) (string, error)
}

// Judge evaluates one claim against its candidates. It never panics and
// never fabricates: any failure lands in UNVERIFIED with a note.
func Judge(client Chatter, claim string, candidates []retrieve.Candidate) Result {
	res := Result{Claim: claim, Status: Unverified, Retrieved: len(candidates)}
	if len(candidates) > 0 {
		res.TopScore = candidates[0].Score
	}

	if len(candidates) == 0 {
		res.Status = Ghost
		res.Reason = "no evidence chunks existed to check against"
		return res
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Claim: %s\n\nEvidence:\n", claim)
	for i, c := range candidates {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, c.Text)
	}

	user := b.String() + "\nReply with exactly three lines: a VERDICT line, an EVIDENCE line with bracketed numbers like [1], [2], and a REASON line."
	reply, err := client.Chat(systemPrompt, user)
	if err == nil && !hasCitation(reply) && verdictRe.MatchString(reply) {
		// One format retry: the judge gave a verdict but no citations.
		// The retry asks for the missing line only; it cannot change the verdict.
		reply2, err2 := client.Chat(systemPrompt, user+"\nYou forgot the EVIDENCE line. Reply again, three lines exactly.")
		if err2 == nil {
			reply = reply2
		}
	}
	if os.Getenv("ECTOCHECK_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] claim=%q\n[debug] raw reply:\n%s\n---\n", claim, reply)
	}
	if err != nil {
		res.Reason = "model call failed: " + err.Error()
		res.ModelError = true
		return res
	}

	m := verdictRe.FindStringSubmatch(strings.ToUpper(reply))
	if m == nil {
		res.Reason = "model reply had no parseable verdict"
		res.ModelError = true
		return res
	}
	res.Status = Status(strings.ToUpper(m[1]))

	// Validate citations: only positions the model was actually shown count.
	// Invented positions flip BadCitation and are dropped, and a verdict of
	// SUPPORTED built on invented citations cannot stand.
	validSeen := map[int]bool{}
	invented := false
	for _, m := range citeRe.FindAllStringSubmatch(reply, -1) {
		pos := 0
		for _, ch := range m[1] {
			pos = pos*10 + int(ch-'0')
		}
		if pos >= 1 && pos <= len(candidates) {
			if !validSeen[pos] {
				validSeen[pos] = true
				res.Cited = append(res.Cited, candidates[pos-1])
			}
		} else {
			invented = true
		}
	}
	res.BadCitation = invented

	if res.Status == Supported && len(res.Cited) == 0 {
		// SUPPORTED with no valid citation is an assertion, not a verdict.
		res.Status = Unverified
		res.Reason = "model claimed support but cited no valid evidence"
		res.BadCitation = true
		return res
	}
	if invented && res.Status != Ghost {
		res.Status = Unverified
		res.Reason = "model cited evidence it was never shown"
		return res
	}

	res.Reason = extractReason(reply)
	return res
}

// hasCitation reports whether the reply contains any bracketed evidence number.
func hasCitation(reply string) bool {
	return citeRe.MatchString(reply)
}

// extractReason pulls the REASON: line if the model produced one.
func extractReason(reply string) string {
	for _, ln := range strings.Split(reply, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(strings.ToUpper(t), "REASON:") {
			return strings.TrimSpace(t[len("REASON:"):])
		}
	}
	return ""
}
