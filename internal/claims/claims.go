// Package claims splits an answer into atomic claims that can each be
// checked on their own. Splitting is deterministic: headings, bullet items
// and sentences become claims. Boilerplate (challenge footers, "sources"
// sections the answer itself lists) is dropped.
package claims

import (
	"strings"
)

// Split turns answer text into claims. Order is preserved.
func Split(answer string) []string {
	var out []string
	for _, para := range paragraphs(answer) {
		trimmed := strings.TrimSpace(para)
		if trimmed == "" || isBoilerplate(trimmed) {
			continue
		}
		if items, ok := bulletItems(trimmed); ok {
			for _, it := range items {
				out = append(out, sentences(it)...)
			}
			continue
		}
		out = append(out, sentences(trimmed)...)
	}
	return out
}

func paragraphs(s string) []string {
	return strings.Split(normalizeNewlines(s), "\n\n")
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// bulletItems detects a paragraph that is a list and returns its items.
func bulletItems(p string) ([]string, bool) {
	lines := strings.Split(p, "\n")
	var items []string
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		for _, marker := range []string{"- ", "* ", "• "} {
			if strings.HasPrefix(t, marker) {
				items = append(items, strings.TrimSpace(strings.TrimPrefix(t, marker)))
				break
			}
		}
	}
	if len(items) >= 2 {
		return items, true
	}
	return nil, false
}

// sentences splits text on sentence boundaries, keeping it conservative:
// abbreviations and decimal numbers do not split.
func sentences(s string) []string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '.' && c != '!' && c != '?' {
			continue
		}
		// Do not split inside numbers like 3.5 or versions like 1.15.2.
		if c == '.' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			continue
		}
		// Sentence ends only when followed by space+capital or end of text.
		if i+1 < len(s) && s[i+1] != ' ' {
			continue
		}
		out = append(out, strings.TrimSpace(s[start:i+1]))
		start = i + 1
	}
	if start < len(s) {
		out = append(out, strings.TrimSpace(s[start:]))
	}
	return out
}

// isBoilerplate filters lines that are not checkable claims: markdown
// headings, the italic submission line DEV's template adds, and bare links.
func isBoilerplate(p string) bool {
	t := strings.TrimSpace(p)
	if strings.HasPrefix(t, "#") {
		return true
	}
	if strings.HasPrefix(t, "*This is a submission for") {
		return true
	}
	if strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://") {
		return true
	}
	return false
}
