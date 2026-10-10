package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// The helper both projects carry, word for word, comment and all: the
// thing people copy from their old project into their new one.
const retryHelper = `
// Servers send Retry-After in seconds or as an HTTP date, and a few send
// both in one header, which no parser in the standard library expects.
func parseRetryAfter(header string, now time.Time) (time.Duration, bool) {
	header = strings.TrimSpace(strings.SplitN(header, ",", 2)[0])
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	if when, err := http.ParseTime(header); err == nil {
		return when.Sub(now), true
	}
	return 0, false
}
`

// twoProjects makes a named project and an anonymous one by the same
// person, who copied a helper and a connection string across and left an
// address in a test.
func twoProjects(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	named, anonymous := filepath.Join(root, "retry"), filepath.Join(root, "fetcher")
	write(t, named, map[string]string{
		"LICENSE":             "MIT License\n\nCopyright (c) 2026 Jane Rook\n\nPermission is hereby granted, free of charge.\n",
		"go.mod":              "module github.com/janerook/retry\n\ngo 1.22\n\nrequire github.com/quietlib/backoffx v0.3.1\n",
		".github/FUNDING.yml": "ko_fi: janerook\n",
		"retry.go":            "package retry\n\nimport (\n\t\"net/http\"\n\t\"strconv\"\n\t\"strings\"\n\t\"time\"\n)\n" + retryHelper + "\nconst dsn = \"postgres://app:hunter2-correct-staple@db.internal/app\"\n",
		"retry_test.go":       "package retry\n\nconst owner = \"rook@fastmail.com\"\n",
	})
	write(t, anonymous, map[string]string{
		"go.mod":            "module github.com/quietfox/fetcher\n\ngo 1.22\n\nrequire github.com/quietlib/backoffx v0.3.1\n",
		"fetch.go":          "package fetcher\n\nimport (\n\t\"net/http\"\n\t\"strconv\"\n\t\"strings\"\n\t\"time\"\n)\n" + retryHelper + "\nconst dsn = \"postgres://app:hunter2-correct-staple@db.internal/app\"\n",
		"fetch_test.go":     "package fetcher\n\n// Reported by rook@fastmail.com when the header had both forms.\nconst sample = \"120, Wed, 21 Oct 2015 07:28:00 GMT\"\n",
		"vendor/x/x.go":     "package x\n",
		"static/app.min.js": strings.Repeat("var a=1;", 1000),
	})
	return named, anonymous
}

func TestTwoProjectsAreTied(t *testing.T) {
	named, anonymous := twoProjects(t)
	report := reportOn(t, true, named, anonymous)
	for _, want := range []string{
		"[NAMES  ] retry and fetcher name the same owner",
		"email address        rook@fastmail.com",
		"[SHARED ]",
		"8 lines, retry.go:10 and fetch.go:10",
		"[DEPENDS]",
		"github.com/quietlib/backoffx",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("the report does not say %q:\n%s", want, report)
		}
	}
	s := study([]*Source{readSource(named), readSource(anonymous)}, false)
	if s.writeUp().worst() != Names {
		t.Error("a shared address should be the loudest thing in the report")
	}
}

// A script copied whole is one copy. The #! line on top used to run into the
// comment under it, and the comment came out again as a finding of its own.
func TestACopiedScriptIsOneCopy(t *testing.T) {
	root := t.TempDir()
	script := "#!/bin/sh\n# Build the programs that get published, one archive per system.\nset -eu\n" +
		"for os in linux darwin windows; do\n\tGOOS=$os go build -trimpath -ldflags '-s -w' -o dist/tool-$os .\ndone\n"
	for _, name := range []string{"one", "two"} {
		write(t, filepath.Join(root, name), map[string]string{"build.sh": script})
	}
	frags := sharedFragments(readSource(filepath.Join(root, "one")), readSource(filepath.Join(root, "two")))
	if len(frags) != 1 || frags[0].Kind != "lines" || frags[0].Lines != 3 {
		t.Errorf("one copied script came out as %d findings: %+v", len(frags), frags)
	}
}

// One folder on its own is described: what in it names its owner.
func TestOneProjectSaysWhoseItIs(t *testing.T) {
	named, _ := twoProjects(t)
	report := reportOn(t, true, named)
	for _, want := range []string{"code host account    janerook", "copyright holder     Jane Rook",
		"funding account      janerook", "email address        rook@fastmail.com"} {
		if !strings.Contains(report, want) {
			t.Errorf("the report does not say %q:\n%s", want, report)
		}
	}
}

// Vendored code, generated code and minified bundles were not written by
// the author, and measuring them would measure somebody else.
func TestWhatNobodyTypedIsLeftOut(t *testing.T) {
	_, anonymous := twoProjects(t)
	s := readSource(anonymous)
	for _, f := range s.Code {
		if strings.HasPrefix(f.Path, "vendor/") || strings.HasSuffix(f.Path, ".min.js") {
			t.Errorf("%s was read as the author's code", f.Path)
		}
	}
	if s.Left["vendored"] != 1 || s.Left["library"]+s.Left["minified"]+s.Left["generated"] != 1 {
		t.Errorf("what was left out was not counted: %v", s.Left)
	}
}

// Nothing may depend on the order a map came out in.
func TestTheAnswerDoesNotMove(t *testing.T) {
	named, anonymous := twoProjects(t)
	first := reportOn(t, true, named, anonymous)
	for range 3 {
		if reportOn(t, true, named, anonymous) != first {
			t.Fatal("the report changed between two runs over the same folders")
		}
	}
}

// writer is a made up programmer: some habits, a handful of words they
// reach for whatever the project, and the randomness of what any one
// project is about.
type writer struct {
	snake, capital bool
	indent         string
	pet            []string
}

var topics = [][]string{
	{"invoice", "customer", "payment", "ledger", "refund", "receipt", "account", "balance"},
	{"sensor", "reading", "window", "sample", "threshold", "device", "channel", "signal"},
	{"article", "author", "draft", "comment", "tag", "feed", "summary", "archive"},
}

func (w writer) name(r *rand.Rand, words []string) string {
	a := w.pet[r.Intn(len(w.pet))]
	b := words[r.Intn(len(words))]
	if w.snake {
		return a + "_" + b
	}
	return a + strings.ToUpper(b[:1]) + b[1:]
}

func (w writer) project(seed int64, words []string, functions int) string {
	r := rand.New(rand.NewSource(seed))
	var b strings.Builder
	for i := 0; i < functions; i++ {
		fn, arg, out := w.name(r, words), w.name(r, words), w.name(r, words)
		note := w.pet[r.Intn(len(w.pet))] + " the " + words[r.Intn(len(words))] + " before anything else"
		if w.capital {
			note = strings.ToUpper(note[:1]) + note[1:] + "."
		}
		fmt.Fprintf(&b, "def %s(%s, items):\n", fn, arg)
		fmt.Fprintf(&b, "%s# %s\n", w.indent, note)
		fmt.Fprintf(&b, "%s%s = []\n", w.indent, out)
		fmt.Fprintf(&b, "%sfor i in range(len(items)):\n", w.indent)
		fmt.Fprintf(&b, "%s%s%s.append(items[i] * %d)\n", w.indent, w.indent, out, r.Intn(9)+2)
		fmt.Fprintf(&b, "%sreturn %s\n\n", w.indent, out)
	}
	return b.String()
}

func wordsFrom(t *testing.T, body string) Words {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, map[string]string{"main.py": body})
	return wordsOf(readSource(dir), "Python")
}

// The arithmetic on writers made up here, which tests that it works at all
// and not how well it works on people: that is the calibration, and its
// numbers are in the README.
func TestTheSameWordsOnADifferentTopic(t *testing.T) {
	ana := writer{snake: true, indent: "    ", pet: []string{"fetch", "tally", "reckon", "cfg"}}
	ben := writer{capital: true, indent: "  ", pet: []string{"load", "count", "compute", "settings"}}

	// Everybody else: ten more writers over every topic, who make the topic
	// words and the plumbing common, the way the corpus does for real code.
	saved := wordTables
	defer func() { wordTables = saved }()
	everybody := &wordTable{df: map[uint64]int{}}
	for i := 0; i < 10; i++ {
		other := writer{indent: "    ", pet: []string{fmt.Sprintf("verb%c", 'a'+i), "make", "get"}}
		for j, topic := range topics {
			everybody.people++
			for k := range wordsFrom(t, other.project(int64(100+10*i+j), topic, 40)).Set {
				everybody.df[hashOf(k)]++
			}
		}
	}
	wordTables = map[string]*wordTable{"Python": everybody}
	same := sharedWords(wordsFrom(t, ana.project(1, topics[0], 150)), wordsFrom(t, ana.project(2, topics[1], 150)))
	other := sharedWords(wordsFrom(t, ana.project(3, topics[0], 150)), wordsFrom(t, ben.project(4, topics[0], 150)))
	if !same.Ready || !other.Ready {
		t.Fatal("not enough code to compare")
	}
	if same.Score <= other.Score {
		t.Errorf("one writer on two topics shared less (%.3f) than two writers on one topic (%.3f)", same.Score, other.Score)
	}
	found := false
	for _, w := range same.Words {
		if strings.HasSuffix(w, ":reckon") || strings.HasSuffix(w, ":tally") {
			found = true
		}
	}
	if !found {
		t.Errorf("the writer's own words are not among what they share: %v", same.Words[:min(10, len(same.Words))])
	}
}

func TestNamesComeApartIntoWords(t *testing.T) {
	for name, want := range map[string]string{
		"parseRetryAfter": "parse retry after",
		"MAX_RETRIES":     "max retries",
		"HTTPServer":      "http server",
		"user_id2name":    "user id name",
	} {
		if got := strings.Join(subwords(name), " "); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}
