package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The README makes promises. Promises in a README rot; promises with a test
// behind them break the build first.

// It only reads. The folders it is pointed at are somebody's projects, and
// a tool that touched them would be a worse problem than anything it found.
func TestNeverWrites(t *testing.T) {
	banned := []string{
		"os.WriteFile", "os.Create", "os.OpenFile", "os.Remove", "os.RemoveAll",
		"os.Rename", "os.Truncate", "os.Chmod", "os.Chown", "os.Mkdir",
		"os.MkdirAll", "os.Symlink", "os.Link", "os.Chtimes", "os.CreateTemp",
		"ioutil.WriteFile",
	}
	scanSource(t, banned, "this tool must not write anything")
}

// No network code at all: not a promise not to send anything, but no way
// to. The quote in front catches every package whose path starts with net,
// so nobody has to take anybody's word for which of them are harmless.
// This matters more here than anywhere: the code this is pointed at is the
// code somebody is trying to keep apart from their name.
func TestNeverOpensTheNetwork(t *testing.T) {
	banned := []string{`"net`, "net.Dial", "http.Get", "http.Post", "syscall.Socket"}
	scanSource(t, banned, "this tool must have no way to send anything anywhere")
}

func TestNeverRunsAnotherProgram(t *testing.T) {
	scanSource(t, []string{`"os/exec"`, "syscall.Exec", "syscall.ForkExec", "StartProcess"},
		"this tool must not run another program")
}

// And there is no model in it. Nothing is downloaded, nothing is trained,
// and every comparison is arithmetic somebody can redo by hand. The tables
// it carries are counts and averages, not weights.
func TestNoModelAnywhere(t *testing.T) {
	banned := []string{"onnx", "gguf", "tensor", "transformers", "safetensors", "embedding"}
	scanSource(t, banned, "the method here is arithmetic and stays that way")
}

func scanSource(t *testing.T, banned []string, why string) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	looked := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		looked++
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range banned {
			if bytes.Contains(src, []byte(bad)) {
				t.Errorf("%s contains %q: %s", name, bad, why)
			}
		}
	}
	if looked == 0 {
		t.Fatal("no source was read, so this test proved nothing")
	}
}

// The long dash is not used anywhere in this project, in code, comments,
// documentation or anything the program prints. Written as a number here
// so this file passes its own test. Binaries and the gzipped tables are
// skipped: they are not text anybody wrote.
func TestNoLongDash(t *testing.T) {
	dash := []byte(string(rune(0x2014)))
	looked := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") && d.Name() != ".github" || d.Name() == "dist") {
				return fs.SkipDir
			}
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(body, 0) >= 0 || strings.HasSuffix(path, ".gz") {
			return nil
		}
		looked++
		if bytes.Contains(body, dash) {
			t.Errorf("%s has a long dash in it", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if looked < 10 {
		t.Fatalf("only %d files were looked at", looked)
	}
}

// Every rule masks exactly one part of what it matches. A rule with no
// group would match and cover nothing.
func TestEveryRuleCoversSomething(t *testing.T) {
	for _, d := range detectors {
		if d.Match.NumSubexp() != 1 {
			t.Errorf("secret rule %q has %d groups", d.Name, d.Match.NumSubexp())
		}
	}
	for _, r := range leafRules {
		if r.re.NumSubexp() != 1 {
			t.Errorf("name rule %q has %d groups", r.kind, r.re.NumSubexp())
		}
	}
}

// Without --reveal the report names nothing, so it can be pasted into an
// issue; with it, the names are there. A secret is never there.
func TestTheMaskedReportNamesNothing(t *testing.T) {
	named, anonymous := twoProjects(t)
	masked := reportOn(t, false, named, anonymous)
	for _, name := range []string{"janerook", "Jane Rook", "rook@fastmail", "parseRetryAfter"} {
		if strings.Contains(masked, name) {
			t.Errorf("the masked report printed %q", name)
		}
	}
	revealed := reportOn(t, true, named, anonymous)
	for _, name := range []string{"rook@fastmail.com", "Servers send Retry-After"} {
		if !strings.Contains(revealed, name) {
			t.Errorf("--reveal did not print %q", name)
		}
	}
	for _, report := range []string{masked, revealed} {
		if strings.Contains(report, "hunter2-correct-staple") {
			t.Error("a secret reached the report")
		}
	}
}

func reportOn(t *testing.T, reveal bool, where ...string) string {
	t.Helper()
	var sources []*Source
	for _, w := range where {
		sources = append(sources, readSource(w))
	}
	var out bytes.Buffer
	study(sources, reveal).writeUp().render(&out)
	return out.String()
}
