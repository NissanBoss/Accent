package main

// The lines two projects have in common.
//
// This is the cheapest way to tie two repositories together and the one
// people forget. A helper function copied from the old project to the new
// one, a comment that explains the same workaround in the same words, an
// error message, a build script: each of them is one search on GitHub away
// from the other repository, and finding them takes no statistics at all.
//
// What makes a shared line worth reporting is that nobody else has it.
// "if err != nil {" is in both projects and in every Go project, so lines
// that turned up in the code of three or more different people in the
// calibration corpus are left out, and so are lines too short or too plain
// to search for.

import (
	"hash/fnv"
	"path"
	"sort"
	"strings"
	"unicode"
)

type Place struct {
	File string
	Line int
}

type Fragment struct {
	// Kind is "lines" for a run of code or configuration, "comment" or
	// "message" for the words inside a comment or a string.
	Kind  string
	Text  string
	Lines int
	A, B  Place
}

// spot is where a normalised piece of text was found.
type spot struct {
	place Place
	raw   string
}

func normalise(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

// searchable says whether a line is long enough and particular enough that
// somebody could search for it and find only you.
func searchable(line string) bool {
	if len(line) < 24 {
		return false
	}
	words := 0
	for _, w := range strings.FieldsFunc(line, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if len(w) >= 3 {
			words++
		}
	}
	return words >= 3
}

func hashOf(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// textsOf gathers, for one source, every searchable line and every comment
// and message with enough words in it, keyed by its normalised text.
func textsOf(s *Source) (lines, comments, messages map[string][]spot) {
	lines, comments, messages = map[string][]spot{}, map[string][]spot{}, map[string][]spot{}
	add := func(into map[string][]spot, key string, p Place, raw string) {
		if common(key) {
			return
		}
		into[key] = append(into[key], spot{p, raw})
	}
	for _, f := range append(append([]*File(nil), s.Code...), s.Other...) {
		for i, raw := range strings.Split(f.Text, "\n") {
			key := normalise(raw)
			if searchable(key) && !importLine(key) {
				add(lines, key, Place{f.Path, i + 1}, strings.TrimSpace(raw))
			}
		}
		if f.Lang == nil {
			continue
		}
		tokens, _ := lex(f.Lang, f.Text)
		for k := 0; k < len(tokens); k++ {
			t := tokens[k]
			switch {
			case t.Kind == Comment && t.Line == 1 && strings.HasPrefix(t.Text, "#!"):
				// The interpreter line on top of a script is nobody's words.
				// Run into the comment under it, it kept that comment from
				// being folded into the block it had been copied with.
			case t.Kind == Comment || t.Kind == Str && t.Doc:
				// A paragraph written one // comment a line is one
				// comment, not five.
				text, last := t.Text, t.Line
				for k+1 < len(tokens) && tokens[k+1].Kind == Comment && tokens[k+1].Own && tokens[k+1].Line == last+1 {
					k++
					text += "\n" + tokens[k].Text
					last = tokens[k].Line
				}
				if body := proseOf(text); len(strings.Fields(body)) >= 6 {
					add(comments, strings.ToLower(body), Place{f.Path, t.Line}, body)
				}
			case t.Kind == Str:
				if body := proseOf(t.Text); len(strings.Fields(body)) >= 4 && len(body) >= 20 {
					add(messages, strings.ToLower(body), Place{f.Path, t.Line}, body)
				}
			}
		}
	}
	return lines, comments, messages
}

// proseOf strips a comment or a string down to its words.
func proseOf(s string) string {
	s = strings.Trim(s, "/*#-\"'`! \t\r\n")
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimLeft(strings.TrimSpace(line), "*/#- ")
		b.WriteString(line)
		b.WriteByte(' ')
	}
	return normalise(b.String())
}

func sharedFragments(a, b *Source) []Fragment {
	la, ca, ma := textsOf(a)
	lb, cb, mb := textsOf(b)
	var out []Fragment

	// Runs of lines: consecutive shared lines in one of A's files are one
	// copied block, and a block of six is worth more than six lines.
	type run struct {
		start, other Place
		end, lines   int
		first        string
	}
	var runs []run
	shared := map[Place]string{}
	for key, spots := range la {
		if _, ok := lb[key]; ok {
			for _, sp := range spots {
				shared[sp.place] = key
			}
		}
	}
	places := make([]Place, 0, len(shared))
	for p := range shared {
		places = append(places, p)
	}
	sort.Slice(places, func(i, j int) bool {
		if places[i].File != places[j].File {
			return places[i].File < places[j].File
		}
		return places[i].Line < places[j].Line
	})
	for i := 0; i < len(places); {
		j := i + 1
		for j < len(places) && places[j].File == places[i].File && places[j].Line-places[j-1].Line <= 2 {
			j++
		}
		key := shared[places[i]]
		runs = append(runs, run{start: places[i], other: lb[key][0].place, end: places[j-1].Line, lines: j - i, first: la[key][0].raw})
		i = j
	}
	// A comment or a message inside a copied block is already in the
	// block, and listing it again would make one copy look like three.
	inBlock := map[Place]bool{}
	for _, r := range runs {
		out = append(out, Fragment{Kind: "lines", Text: r.first, Lines: r.lines, A: r.start, B: r.other})
		if r.lines >= 2 {
			for line := r.start.Line; line <= r.end; line++ {
				inBlock[Place{r.start.File, line}] = true
			}
		}
	}
	for _, kind := range []struct {
		name   string
		mine   map[string][]spot
		theirs map[string][]spot
	}{{"comment", ca, cb}, {"message", ma, mb}} {
		for key, spots := range kind.mine {
			if other, ok := kind.theirs[key]; ok && !inBlock[spots[0].place] {
				out = append(out, Fragment{Kind: kind.name, Text: spots[0].raw, A: spots[0].place, B: other[0].place})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		wi, wj := fragmentWeight(out[i]), fragmentWeight(out[j])
		if wi != wj {
			return wi > wj
		}
		if out[i].A.File != out[j].A.File {
			return out[i].A.File < out[j].A.File
		}
		return out[i].A.Line < out[j].A.Line
	})
	return out
}

// fragmentWeight orders what was found: comments and messages are words
// somebody chose, and a block of copied lines beats a single line.
func fragmentWeight(f Fragment) int {
	switch f.Kind {
	case "comment":
		return 1000 + len(f.Text)
	case "message":
		return 500 + len(f.Text)
	}
	return f.Lines*100 + len(f.Text)
}

// common says whether a line, a comment or a message turned up in the code
// of three or more different people in the calibration corpus.
func common(key string) bool {
	return commonTexts[hashOf(key)]
}

// importLine says whether a line only brings a library in. Two projects
// importing the same thing is what dependencies are for, and deps.go
// weighs it against how many people use that library; as a shared line it
// would count every import of a popular package as a clue.
func importLine(line string) bool {
	for _, p := range []string{"import ", "from ", "#include", "using ", "use ", "require ", "package ",
		"extern crate ", "@import"} {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return strings.Contains(line, "require(") && (strings.HasPrefix(line, "const ") ||
		strings.HasPrefix(line, "var ") || strings.HasPrefix(line, "let "))
}

// templated says whether a file is mostly what a tool or a template wrote:
// Android layouts, build manifests, dependency lists, licences, ignore
// files. Two strangers' projects share lines in these all the time, and
// the calibration found them behind most of the lines strangers shared.
func templated(rel string) bool {
	base := strings.ToLower(path.Base(rel))
	switch path.Ext(base) {
	case ".xml", ".properties", ".gradle", ".json", ".lock", ".sum", ".csproj", ".sln", ".plist":
		return true
	}
	switch base {
	case "go.mod", "cargo.toml", "requirements.txt", "pipfile", "setup.cfg", ".gitignore",
		".dockerignore", ".editorconfig", "gradlew", "gradlew.bat", "mvnw", "mvnw.cmd":
		return true
	}
	return strings.HasPrefix(base, "license") || strings.HasPrefix(base, "licence") ||
		strings.HasPrefix(base, "copying") || strings.HasPrefix(base, "notice")
}
