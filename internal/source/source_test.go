package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadChunksFilesAndDirs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("first line\nsecond line\n\nthird para\n"), 0o644)
	os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("other text\n"), 0o644)
	os.Mkdir(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, ".git", "ignored"), []byte("secret\n"), 0o644)

	chunks, err := Load([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("no chunks")
	}
	joined := ""
	for _, c := range chunks {
		joined += c.Text + "\n"
		if !strings.HasPrefix(c.ID, "E") {
			t.Errorf("bad ID %q", c.ID)
		}
		if c.File == "" || c.Line < 1 {
			t.Errorf("bad location %+v", c)
		}
	}
	if !strings.Contains(joined, "first line") || !strings.Contains(joined, "other text") {
		t.Errorf("missing content: %q", joined)
	}
	if strings.Contains(joined, "secret") {
		t.Error("hidden .git directory was loaded")
	}
}

func TestIDsAreSequential(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n\ntwo\n\nthree\n"), 0o644)
	chunks, err := Load([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range chunks {
		want := "E" + itoa(i+1)
		if c.ID != want {
			t.Errorf("chunk %d: got ID %q want %q", i, c.ID, want)
		}
	}
}

func TestLongLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("x", 600)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte(long+"\nshort line\n"), 0o644)
	chunks, err := Load([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if strings.Contains(c.Text, "xxx") {
			t.Error("over-long line was kept")
		}
	}
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(chunks))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
