// Package source loads local files and turns them into numbered evidence
// chunks. Every chunk gets a stable ID (E1, E2, ...) that the rest of the
// pipeline and the report cite. IDs are assigned here and only here, so an
// ID appearing anywhere else can always be traced back to real text.
package source

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Chunk is one addressable piece of ground truth.
type Chunk struct {
	ID   string // E1, E2, ...
	File string // path it came from
	Line int    // 1-based line number of the first line
	Text string
}

// Load reads every path (files or directories, recursed) and chunks their
// text. Binary-looking files and very long lines are skipped. Empty input
// yields an empty slice, not an error.
func Load(paths []string) ([]Chunk, error) {
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && path != p {
					return filepath.SkipDir // skip .git and friends
				}
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	var chunks []Chunk
	n := 0
	for _, f := range files {
		part, err := chunkFile(f, &n)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, part...)
	}
	return chunks, nil
}

const (
	maxLineLen  = 500
	maxChunkRun = 4 // lines merged into one chunk
)

func chunkFile(path string, n *int) ([]Chunk, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer f.Close()

	var chunks []Chunk
	var run []string
	runStart := 0
	lineNo := 0

	flush := func() {
		if len(run) == 0 {
			return
		}
		*n++
		chunks = append(chunks, Chunk{
			ID:   fmt.Sprintf("E%d", *n),
			File: path,
			Line: runStart,
			Text: strings.Join(run, "\n"),
		})
		run = nil
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if len(line) > maxLineLen {
			flush()
			continue // minified/generated line; not citable evidence
		}
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if looksBinary(line) {
			flush()
			continue
		}
		if len(run) == 0 {
			runStart = lineNo
		}
		run = append(run, line)
		if len(run) >= maxChunkRun {
			flush()
		}
	}
	flush()
	return chunks, sc.Err()
}

func looksBinary(s string) bool {
	for _, r := range s {
		if r == 0 {
			return true
		}
	}
	return false
}
