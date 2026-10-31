//go:build integration

// Run with: ECTOCHECK_INTEGRATION=1 go test -tags integration -run Integration -v
// Needs `ollama serve` with qwen2.5:0.5b on 127.0.0.1:11434.
package main

import (
	"os"
	"testing"

	"github.com/DhanushNehru/ectocheck/internal/claims"
	"github.com/DhanushNehru/ectocheck/internal/ollama"
	"github.com/DhanushNehru/ectocheck/internal/retrieve"
	"github.com/DhanushNehru/ectocheck/internal/source"
	"github.com/DhanushNehru/ectocheck/internal/verdict"
)

func TestIntegrationRealModel(t *testing.T) {
	if os.Getenv("ECTOCHECK_INTEGRATION") == "" {
		t.Skip("set ECTOCHECK_INTEGRATION=1")
	}
	evidence, err := source.Load([]string{"examples/homeserver"})
	if err != nil {
		t.Fatal(err)
	}
	answer, _ := os.ReadFile("examples/answer.md")
	client := ollama.New("http://127.0.0.1:11434", "qwen2.5:0.5b")
	for _, c := range claims.Split(string(answer)) {
		res := verdict.Judge(client, c, retrieve.Top(string(answer), c, evidence, 3))
		t.Logf("%-10s %s (%s)", res.Status, c, res.Reason)
	}
}
