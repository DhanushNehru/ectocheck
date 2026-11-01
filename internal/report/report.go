// Package report renders the hunt. Text output is meant for humans and
// README screenshots; JSON output is for CI.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/DhanushNehru/ectocheck/internal/verdict"
)

// Summary counts verdicts.
type Summary struct {
	Claims     int `json:"claims"`
	Supported  int `json:"supported"`
	Partial    int `json:"partial"`
	Ghosts     int `json:"ghosts"`
	Unverified int `json:"unverified"`
}

// Summarize tallies results.
func Summarize(results []verdict.Result) Summary {
	s := Summary{Claims: len(results)}
	for _, r := range results {
		switch r.Status {
		case verdict.Supported:
			s.Supported++
		case verdict.Partial:
			s.Partial++
		case verdict.Ghost:
			s.Ghosts++
		default:
			s.Unverified++
		}
	}
	return s
}

var icon = map[verdict.Status]string{
	verdict.Supported:  "✓",
	verdict.Partial:    "~",
	verdict.Ghost:      "👻",
	verdict.Unverified: "?",
}

// WriteText renders the human report.
func WriteText(w io.Writer, results []verdict.Result, s Summary) {
	for i, r := range results {
		fmt.Fprintf(w, "%s %-10s %s\n", icon[r.Status], string(r.Status), r.Claim)
		if r.Reason != "" {
			fmt.Fprintf(w, "  %12s%s\n", "", r.Reason)
		}
		for _, c := range r.Cited {
			oneLine := strings.Join(strings.Fields(c.Text), " ")
			if len(oneLine) > 90 {
				oneLine = oneLine[:90] + "…"
			}
			fmt.Fprintf(w, "  %12s[%s] %s:%d — %s\n", "", c.ID, c.File, c.Line, oneLine)
		}
		if r.BadCitation {
			fmt.Fprintf(w, "  %12s(note: the judge cited evidence it was never shown; verdict not trusted)\n", "")
		}
		if i < len(results)-1 {
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintf(w, "\n%d claims: %d supported, %d partial, %d ghosts, %d unverified\n",
		s.Claims, s.Supported, s.Partial, s.Ghosts, s.Unverified)
	if s.Ghosts > 0 {
		fmt.Fprintln(w, "The answer is haunted.")
	} else if s.Unverified > 0 {
		fmt.Fprintln(w, "No ghosts, but not every claim could be trusted. Treat unverified claims as unchecked.")
	} else {
		fmt.Fprintln(w, "No ghosts found.")
	}
}

type jsonResult struct {
	Claim  string   `json:"claim"`
	Status string   `json:"status"`
	Reason string   `json:"reason,omitempty"`
	Cited  []string `json:"cited,omitempty"`
}

// WriteJSON renders the machine report.
func WriteJSON(w io.Writer, results []verdict.Result, s Summary) error {
	out := struct {
		Summary Summary      `json:"summary"`
		Claims  []jsonResult `json:"claims"`
	}{Summary: s}
	for _, r := range results {
		jr := jsonResult{Claim: r.Claim, Status: string(r.Status), Reason: r.Reason}
		for _, c := range r.Cited {
			jr.Cited = append(jr.Cited, c.ID)
		}
		out.Claims = append(out.Claims, jr)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
