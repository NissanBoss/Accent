package main

// Turning what was read into what is worth saying, and the words for it.

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type Study struct {
	Sources  []*Source
	Profiles []map[string]*Profile
	Leaves   [][]Leaf
	Deps     [][]Dep
	reveal   bool
}

func study(sources []*Source, reveal bool) *Study {
	s := &Study{Sources: sources, reveal: reveal}
	for _, src := range sources {
		s.Profiles = append(s.Profiles, profilesOf(src))
		s.Leaves = append(s.Leaves, distinctLeaves(leavesOf(src)))
		s.Deps = append(s.Deps, depsOf(src))
	}
	return s
}

func (s *Study) read() int {
	n := 0
	for _, src := range s.Sources {
		n += len(src.Code)
	}
	return n
}

func (s *Study) writeUp() *Report {
	var names []string
	for _, src := range s.Sources {
		names = append(names, src.Name)
	}
	r := &Report{Title: "Accent: " + strings.Join(names, ", ")}
	for i, src := range s.Sources {
		r.Summary = append(r.Summary, s.summary(i)...)
		r.Gaps = append(r.Gaps, src.Gaps...)
	}
	if len(s.Sources) == 1 {
		s.alone(r)
		return r
	}
	for i := range s.Sources {
		for j := i + 1; j < len(s.Sources); j++ {
			s.pair(r, i, j)
		}
	}
	return r
}

func (s *Study) summary(i int) []string {
	src := s.Sources[i]
	if len(src.Code) == 0 {
		return []string{fmt.Sprintf("%s: no code found.", src.Name)}
	}
	langs := make([]string, 0, len(s.Profiles[i]))
	for name := range s.Profiles[i] {
		langs = append(langs, name)
	}
	sort.Slice(langs, func(a, b int) bool {
		pa, pb := s.Profiles[i][langs[a]], s.Profiles[i][langs[b]]
		if pa.Tokens != pb.Tokens {
			return pa.Tokens > pb.Tokens
		}
		return langs[a] < langs[b]
	})
	var parts []string
	for _, l := range langs {
		p := s.Profiles[i][l]
		parts = append(parts, fmt.Sprintf("%s %s", l, plural(p.Files, "file", "files")))
	}
	line := fmt.Sprintf("%s: %s read (%s).", src.Name, plural(len(src.Code), "file of code", "files of code"),
		strings.Join(parts, ", "))
	lines := wrapText(line, wrap+12)
	if len(src.Left) > 0 {
		reasons := make([]string, 0, len(src.Left))
		for why, n := range src.Left {
			phrase := leftOut[why]
			reasons = append(reasons, plural(n, phrase[0], phrase[1]))
		}
		sort.Strings(reasons)
		lines = append(lines, wrapText("Left out as not written by hand: "+strings.Join(reasons, ", ")+".", wrap+12)...)
	}
	return lines
}

// alone is the report on one folder: what in it names whoever wrote it.
func (s *Study) alone(r *Report) {
	src := s.Sources[0]
	if leaves := s.Leaves[0]; len(leaves) > 0 {
		var entries [][]string
		for _, l := range leaves {
			entries = append(entries, []string{fmt.Sprintf("%-20s %s", l.Kind, s.show(l.Kind, l.Value)),
				fmt.Sprintf("%-20s %s:%d", "", l.Where.File, l.Where.Line)})
		}
		r.add(Finding{
			Tag: "NAMES  ", Weight: Hints,
			Title: fmt.Sprintf("What in %s says whose it is", src.Name),
			Lines: atMost(entries, 15),
			Advice: "None of this is a leak in a project published under your own name. In one that is " +
				"not, each is a search away from you, and a match ties the two with no room for doubt.",
		})
	}
	ready := false
	langs := make([]string, 0, len(s.Profiles[0]))
	for lang := range s.Profiles[0] {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	for _, lang := range langs {
		p := s.Profiles[0][lang]
		if p.Tokens < minTokens {
			continue
		}
		ready = true
		if odd := unusual(p, 8); len(odd) > 0 {
			r.add(Finding{
				Tag: "HABITS ", Weight: Hints,
				Title: fmt.Sprintf("What sets the %s in %s apart from how most people write it", lang, src.Name),
				Lines: odd,
				Advice: "Measured against the people in the calibration corpus who write " + lang + ". " +
					"These are what makes a second project recognisably yours, and what an attribution " +
					"model reads first. Spacing a formatter decides is left out, because running the " +
					"formatter is the whole cure for it.",
			})
		}
	}
	advice := "Give Accent a second folder, your own named project next to this one, and it says " +
		"what ties them: the names they carry, the lines they share, and the rare words both use."
	if !ready {
		advice = "There is not enough code here to describe a style: about " + commas(minTokens) +
			" tokens in one language are needed. Shared lines and names can still be compared."
	}
	r.add(Finding{Tag: "NOTE   ", Weight: Note, Title: "One folder is described, not compared", Advice: advice})
}

func (s *Study) show(kind, value string) string {
	value = coverAll(value)
	if s.reveal {
		return value
	}
	return maskValue(kind, value)
}

func (s *Study) pair(r *Report, i, j int) {
	a, b := s.Sources[i], s.Sources[j]
	s.sharedNames(r, a, b, s.Leaves[i], s.Leaves[j])
	s.sharedLines(r, a, b)
	s.words(r, a, b)
	if deps := sharedDeps(s.Deps[i], s.Deps[j]); len(deps) > 0 {
		var lines []string
		for _, d := range deps {
			lines = append(lines, fmt.Sprintf("%-10s %s", d.Ecosystem, d.Name))
		}
		r.add(Finding{
			Tag: "DEPENDS", Weight: Hints,
			Title: fmt.Sprintf("%s and %s both use %s few other projects do", a.Name, b.Name,
				some(len(deps), "a library", "libraries")),
			Lines: atMost([][]string{lines}, 1),
			Advice: "A taste in libraries is a habit like any other. On its own it settles nothing; " +
				"next to anything else here, a search of who depends on these lists both projects.",
		})
	}
}

func (s *Study) sharedNames(r *Report, a, b *Source, la, lb []Leaf) {
	in := map[string]Leaf{}
	for _, l := range lb {
		in[l.Kind+"\x00"+strings.ToLower(l.Value)] = l
	}
	var entries [][]string
	for _, l := range la {
		other, ok := in[l.Kind+"\x00"+strings.ToLower(l.Value)]
		if !ok {
			continue
		}
		entries = append(entries, []string{
			fmt.Sprintf("%-20s %s", l.Kind, s.show(l.Kind, l.Value)),
			fmt.Sprintf("%-20s %s:%d  and  %s:%d", "", l.Where.File, l.Where.Line, other.Where.File, other.Where.Line),
		})
	}
	if len(entries) == 0 {
		return
	}
	r.add(Finding{
		Tag: "NAMES  ", Weight: Names,
		Title: fmt.Sprintf("%s and %s name the same owner%s", a.Name, b.Name, ways(len(entries))),
		Lines: atMost(entries, 12),
		Advice: "Nothing else in this report matters next to these. Each is in both projects, and " +
			"a search for any of them finds both. Take them out of the one that is meant to be " +
			"anonymous: the module path, the licence, the funding file, the address in a test.",
	})
}

func (s *Study) sharedLines(r *Report, a, b *Source) {
	frags := sharedFragments(a, b)
	if len(frags) == 0 {
		return
	}
	weight := Hints
	var entries [][]string
	for _, f := range frags {
		if f.Kind != "lines" || f.Lines >= blockLines {
			weight = Ties
		}
		what := map[string]string{"comment": "a comment", "message": "a message"}[f.Kind]
		if f.Kind == "lines" {
			what = some(f.Lines, "a line", "lines")
		}
		text := coverAll(f.Text)
		if !s.reveal {
			text = maskText(text)
		}
		entries = append(entries, []string{
			fmt.Sprintf("%s, %s:%d and %s:%d", what, f.A.File, f.A.Line, f.B.File, f.B.Line),
			"    " + clip(text, wrap-4),
		})
	}
	r.add(Finding{
		Tag: "SHARED ", Weight: weight,
		Title: fmt.Sprintf("%s and %s share %s nobody else has", a.Name, b.Name,
			some(len(entries), "a piece of text", "pieces of text")),
		Lines: atMost(entries, 12),
		Advice: "Each of these is one search away: paste it into GitHub's code search and both " +
			"projects come up. In the one meant to be anonymous, write them again rather than copy " +
			"them, comments in other words and helpers from scratch.",
	})
}

func languageNamed(name string) *Language {
	for _, l := range languages {
		if l.Name == name {
			return l
		}
	}
	return &Language{Name: name}
}

// leftOut is how each reason for not reading something is said, one and
// many.
var leftOut = map[string][2]string{
	"vendored":  {"folder of other people's or built code", "folders of other people's or built code"},
	"generated": {"generated file", "generated files"},
	"minified":  {"minified file", "minified files"},
	"library":   {"copy of a well known library", "copies of well known libraries"},
	"big":       {"file over a megabyte", "files over a megabyte"},
	"binary":    {"binary file", "binary files"},
}

func ways(n int) string {
	if n == 1 {
		return ""
	}
	return fmt.Sprintf(", in %d ways", n)
}

// The bands are where the verdicts live, so they sit here in plain sight.
// They are measured in the calibration, not picked.
const (
	// blockLines is how many lines in a row two projects must share for
	// it to count as copied code. In the corpus a single shared line
	// turned up between strangers' projects one time in five, two together
	// one time in eighteen, three together one time in eighty: mostly a
	// library or a tutorial both had pasted in.
	blockLines = 3
	// wordsSame, wordsClose and wordsHint are how far up the strangers two
	// projects must sit: the share of pairs of strangers in the corpus that
	// share less of their rare vocabulary than these two do. Only wordsSame
	// is a tie. It catches a third of one person's pairs and about one pair
	// of strangers in a hundred; wordsClose catches half of one person's
	// pairs, and would call one pair of strangers in twenty tied.
	wordsSame  = 0.99
	wordsClose = 0.95
	wordsHint  = 0.80
)

func (s *Study) words(r *Report, a, b *Source) {
	var langs []string
	ta, tb := map[string]Words{}, map[string]Words{}
	for _, f := range a.Code {
		if _, ok := ta[f.Lang.Name]; !ok {
			ta[f.Lang.Name] = wordsOf(a, f.Lang.Name)
		}
	}
	for _, f := range b.Code {
		if _, ok := tb[f.Lang.Name]; !ok {
			tb[f.Lang.Name] = wordsOf(b, f.Lang.Name)
		}
	}
	for lang, wa := range ta {
		if wb, ok := tb[lang]; ok && wa.Tokens >= minTokens && wb.Tokens >= minTokens {
			langs = append(langs, lang)
		}
	}
	sort.Strings(langs)
	if len(langs) == 0 {
		r.add(Finding{
			Tag: "WORDS  ", Weight: Note,
			Title: fmt.Sprintf("Not enough code in one language to compare the words of %s and %s", a.Name, b.Name),
			Advice: "Words are compared within one language, and each side needs about " + commas(minTokens) +
				" tokens of it. The names and lines both carry are still compared.",
		})
		return
	}
	for _, lang := range langs {
		sh := sharedWords(ta[lang], tb[lang])
		if !sh.Ready {
			continue
		}
		f := Finding{Tag: "WORDS  "}
		closer := fmt.Sprintf("closer than %.0f pairs of strangers in 100", math.Floor(sh.Closer*100))
		if sh.Closer >= 0.995 {
			closer = "closer than any two strangers in the corpus"
		}
		switch {
		case sh.Closer >= wordsSame:
			f.Weight = Ties
			f.Title = fmt.Sprintf("The %s in %s and %s uses the same rare words as one person would: %s",
				lang, a.Name, b.Name, closer)
		case sh.Closer >= wordsClose:
			f.Weight = Hints
			f.Title = fmt.Sprintf("The %s in %s and %s shares more rare words than nearly all strangers: %s",
				lang, a.Name, b.Name, closer)
		case sh.Closer >= wordsHint:
			f.Weight = Hints
			f.Title = fmt.Sprintf("The %s in %s and %s shares more rare words than most strangers: %s",
				lang, a.Name, b.Name, closer)
		default:
			f.Weight = Note
			f.Title = fmt.Sprintf("Nothing in the words of the %s in %s and %s ties them", lang, a.Name, b.Name)
		}
		if f.Weight >= Hints && len(sh.Words) > 0 {
			f.Lines = append(f.Lines, fmt.Sprintf("%s few people use, rarest first:", some(len(sh.Words), "a word", "words")))
			var entries [][]string
			for _, w := range sh.Words {
				kind, word, _ := strings.Cut(w, ":")
				if !s.reveal {
					word = maskText(word)
				}
				entries = append(entries, []string{fmt.Sprintf("  %-12s %s", wordKinds[kind], clip(word, wrap-16))})
			}
			f.Lines = append(f.Lines, atMost(entries, 12)...)
			f.Advice = "These are words, and words are what an attribution model reads first and " +
				"a search finds fastest. In the project meant to be anonymous, name things differently " +
				"and write the comments in other words; a formatter will not do it for you."
		}
		r.add(f)
	}
}

// wordKinds says where a shared word came from.
var wordKinds = map[string]string{
	"name": "a name", "short": "a short name", "part": "in names", "parts": "in names",
	"said": "in comments", "saids": "in comments", "message": "in messages",
}
