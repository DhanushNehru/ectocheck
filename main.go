// ectocheck hunts ghosts in AI-generated answers.
//
// A "ghost" is a claim that cannot point to real evidence in the local
// source files you provide. ectocheck splits an answer into claims,
// retrieves candidate evidence from your sources, asks a small local
// model (via Ollama) for a verdict on each claim, and validates every
// citation the model produces. Citations to evidence that was never
// retrieved are treated as invented and the claim is not trusted.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/DhanushNehru/ectocheck/internal/claims"
	"github.com/DhanushNehru/ectocheck/internal/ollama"
	"github.com/DhanushNehru/ectocheck/internal/report"
	"github.com/DhanushNehru/ectocheck/internal/retrieve"
	"github.com/DhanushNehru/ectocheck/internal/source"
	"github.com/DhanushNehru/ectocheck/internal/verdict"
)

func main() {
	var (
		srcFlags multiFlag
		model    = flag.String("model", "qwen2.5:1.5b", "Ollama model used to judge claims")
		host     = flag.String("ollama", "http://127.0.0.1:11434", "Ollama base URL")
		topK     = flag.Int("top-k", 3, "candidate evidence chunks retrieved per claim")
		format   = flag.String("format", "text", "report format: text or json")
	)
	flag.Var(&srcFlags, "src", "source file or directory with ground truth (repeatable)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ectocheck check <answer-file> --src <file-or-dir> [--src ...] [flags]\n\n")
		flag.PrintDefaults()
	}

	// The standard flag parser stops at the first positional argument, but
	// the natural CLI shape puts flags after the subcommand and file. Split
	// flags from positionals by hand, then parse.
	args := parseInterleaved(os.Args[1:])
	if len(args) != 2 || args[0] != "check" {
		flag.Usage()
		os.Exit(2)
	}
	if len(srcFlags) == 0 {
		fmt.Fprintln(os.Stderr, "ectocheck: at least one --src is required (the ground truth to check against)")
		os.Exit(2)
	}

	answerBytes, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ectocheck: reading answer: %v\n", err)
		os.Exit(2)
	}

	evidence, err := source.Load(srcFlags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ectocheck: loading sources: %v\n", err)
		os.Exit(2)
	}
	if len(evidence) == 0 {
		fmt.Fprintln(os.Stderr, "ectocheck: sources contained no usable text")
		os.Exit(2)
	}

	claimList := claims.Split(string(answerBytes))
	if len(claimList) == 0 {
		fmt.Fprintln(os.Stderr, "ectocheck: no claims found in the answer")
		os.Exit(2)
	}

	client := ollama.New(*host, *model)
	results := make([]verdict.Result, 0, len(claimList))
	for _, c := range claimList {
		candidates := retrieve.Top(string(answerBytes), c, evidence, *topK)
		res := verdict.Judge(client, c, candidates)
		results = append(results, res)
	}

	summary := report.Summarize(results)
	switch strings.ToLower(*format) {
	case "json":
		if err := report.WriteJSON(os.Stdout, results, summary); err != nil {
			fmt.Fprintf(os.Stderr, "ectocheck: writing report: %v\n", err)
			os.Exit(2)
		}
	default:
		report.WriteText(os.Stdout, results, summary)
	}

	// Exit 1 when anything is untrusted (ghosts or unverified claims), so
	// ectocheck can gate CI pipelines on "every claim checked out".
	if summary.Ghosts > 0 || summary.Unverified > 0 {
		os.Exit(1)
	}
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// parseInterleaved separates flag tokens from positional arguments so flags
// may appear anywhere on the command line. It reparses the flags through the
// standard FlagSet and returns only the positionals.
func parseInterleaved(argv []string) []string {
	valueFlags := map[string]bool{
		"-src": true, "--src": true,
		"-model": true, "--model": true,
		"-ollama": true, "--ollama": true,
		"-top-k": true, "--top-k": true,
		"-format": true, "--format": true,
	}
	var flagArgs, positional []string
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
			if valueFlags[a] && !strings.Contains(a, "=") && i+1 < len(argv) {
				i++
				flagArgs = append(flagArgs, argv[i])
			}
			continue
		}
		positional = append(positional, a)
	}
	if err := flag.CommandLine.Parse(flagArgs); err != nil {
		os.Exit(2)
	}
	return positional
}
