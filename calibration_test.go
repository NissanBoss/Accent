package main

// The calibration, against real code by real people rather than the made
// up samples in the other tests, and the tables in data/ that come out of
// it.
//
// The corpus is people on GitHub and the repositories they wrote alone, two
// or three each for nearly all of them, in one language, before coding
// assistants were common. It is other
// people's code, so it is not in this repository, and these tests only
// count: nothing about any one author is printed. Point them at a folder
// laid out as language/author/repository:
//
//	ACCENT_CAL=folder go test -run Calibration -v -timeout 60m
//	ACCENT_CAL=folder ACCENT_BUILD=1 go test -run BuildTables -v -timeout 60m
//
// The second one rebuilds data/ and is the only code in the repository
// that writes a file. It is a test so that it never ships in the binary.
//
// Two repositories by the same author are the same hands on different
// projects, which is the case that matters: nobody publishes their
// anonymous project as a copy of their named one.

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var calibrationLanguages = []struct{ dir, name string }{
	{"go", "Go"}, {"python", "Python"}, {"javascript", "JavaScript"}, {"typescript", "TypeScript"},
	{"rust", "Rust"}, {"java", "Java"},
}

type corpusAuthor struct {
	lang     string
	name     string
	sources  []*Source
	profiles []*Profile
	// words is the vocabulary of each repository with enough code, largest
	// first.
	words []Words
	// What the author's repositories carry, for the tables of what is
	// everybody's: each author counts once however many repositories
	// they have.
	texts, leaves, deps map[string]bool
}

var corpus []*corpusAuthor

func readCorpus(t *testing.T) []*corpusAuthor {
	root := os.Getenv("ACCENT_CAL")
	if root == "" {
		t.Skip("ACCENT_CAL is not set")
	}
	if corpus != nil {
		return corpus
	}
	// The tables must be empty while the corpus is read, or what is common
	// in it would already be missing from what is counted.
	saved := [3]map[uint64]bool{commonTexts, commonLeaves, commonDeps}
	commonTexts, commonLeaves, commonDeps = map[uint64]bool{}, map[uint64]bool{}, map[uint64]bool{}
	defer func() { commonTexts, commonLeaves, commonDeps = saved[0], saved[1], saved[2] }()

	for _, l := range calibrationLanguages {
		owners, _ := os.ReadDir(filepath.Join(root, l.dir))
		for _, o := range owners {
			if !o.IsDir() {
				continue
			}
			a := &corpusAuthor{lang: l.name, name: o.Name(), texts: map[string]bool{}, leaves: map[string]bool{}, deps: map[string]bool{}}
			repos, _ := os.ReadDir(filepath.Join(root, l.dir, o.Name()))
			type sized struct {
				s *Source
				p *Profile
				w Words
			}
			var got []sized
			for _, r := range repos {
				if !r.IsDir() {
					continue
				}
				s := readSource(filepath.Join(root, l.dir, o.Name(), r.Name()))
				lines, comments, messages := textsOf(s)
				for _, m := range []map[string][]spot{lines, comments, messages} {
					for k := range m {
						a.texts[k] = true
					}
				}
				for _, lf := range distinctLeaves(leavesOf(s)) {
					a.leaves[lf.Kind+"\x00"+strings.ToLower(lf.Value)] = true
				}
				for _, d := range depsOf(s) {
					a.deps[d.Ecosystem+"\x00"+d.Name] = true
				}
				if w := wordsOf(s, l.name); w.Tokens >= minTokens {
					got = append(got, sized{s, profilesOf(s)[l.name], w})
				}
			}
			sort.SliceStable(got, func(i, j int) bool { return got[i].w.Tokens > got[j].w.Tokens })
			for i, g := range got {
				// Only the two largest are kept whole: the corpus is gigabytes
				// of other people's code, and the comparisons only ever pair
				// an author's first project with somebody's second.
				if i < 2 {
					a.sources = append(a.sources, g.s)
				}
				a.profiles = append(a.profiles, g.p)
				a.words = append(a.words, g.w)
			}
			corpus = append(corpus, a)
		}
	}
	return corpus
}

// everybodys is what three or more of the given authors have.
func everybodys(authors []*corpusAuthor, pick func(*corpusAuthor) map[string]bool) map[string]bool {
	count := map[string]int{}
	for _, a := range authors {
		for k := range pick(a) {
			count[k]++
		}
	}
	out := map[string]bool{}
	for k, n := range count {
		if n >= 3 {
			out[k] = true
		}
	}
	return out
}

func hashed(keys map[string]bool) map[uint64]bool {
	out := map[uint64]bool{}
	for k := range keys {
		out[hashOf(k)] = true
	}
	return out
}

func halfOf(name string) int {
	h := fnv.New32a()
	h.Write([]byte(name))
	return int(h.Sum32() % 2)
}

// TestCalibrationSharing asks how often two projects share names, lines
// and libraries when one person wrote both, and when two people did. The
// tables of what is everybody's are built from one half of the authors and
// the pairs are taken from the other, so no pair is judged against a table
// it helped build.
func TestCalibrationSharing(t *testing.T) {
	authors := readCorpus(t)
	var background, test []*corpusAuthor
	for _, a := range authors {
		if halfOf(a.name) == 0 {
			background = append(background, a)
		} else if len(a.sources) >= 2 {
			test = append(test, a)
		}
	}
	saved := [3]map[uint64]bool{commonTexts, commonLeaves, commonDeps}
	commonTexts = hashed(everybodys(background, func(a *corpusAuthor) map[string]bool { return a.texts }))
	commonLeaves = hashed(everybodys(background, func(a *corpusAuthor) map[string]bool { return a.leaves }))
	commonDeps = hashed(everybodys(background, func(a *corpusAuthor) map[string]bool { return a.deps }))
	defer func() { commonTexts, commonLeaves, commonDeps = saved[0], saved[1], saved[2] }()

	type tally struct{ pairs, names, blocks, comments, messages, lines, deps int }
	count := func(x, y *Source, into *tally) {
		into.pairs++
		if len(sharedLeavesOf(distinctLeaves(leavesOf(x)), distinctLeaves(leavesOf(y)))) > 0 {
			into.names++
		}
		kinds := map[string]bool{}
		for _, f := range sharedFragments(x, y) {
			switch {
			case f.Kind == "lines" && f.Lines >= blockLines:
				kinds["block"] = true
			default:
				kinds[f.Kind] = true
			}
		}
		for k, v := range map[string]*int{"block": &into.blocks, "comment": &into.comments, "message": &into.messages, "lines": &into.lines} {
			if kinds[k] {
				*v++
			}
		}
		if len(sharedDeps(depsOf(x), depsOf(y))) > 0 {
			into.deps++
		}
	}
	var same, different tally
	for i, a := range test {
		count(a.sources[0], a.sources[1], &same)
		for j := 1; j <= 3; j++ {
			b := test[(i+j)%len(test)]
			if b != a && b.lang == a.lang {
				count(a.sources[0], b.sources[1], &different)
			}
		}
	}
	show := func(name string, x tally) {
		pc := func(n int) string { return fmt.Sprintf("%5.1f%%", 100*float64(n)/math.Max(1, float64(x.pairs))) }
		t.Logf("%-9s %4d pairs: names %s, copied blocks %s, comments %s, messages %s, single lines %s, libraries %s",
			name, x.pairs, pc(x.names), pc(x.blocks), pc(x.comments), pc(x.messages), pc(x.lines), pc(x.deps))
	}
	show("same", same)
	show("different", different)
}

// TestBuildTables writes data/ from the whole corpus.
func TestBuildTables(t *testing.T) {
	if os.Getenv("ACCENT_BUILD") == "" {
		t.Skip("ACCENT_BUILD is not set")
	}
	authors := readCorpus(t)
	writeHashes(t, "data/common-texts.tsv.gz", everybodys(authors, func(a *corpusAuthor) map[string]bool { return a.texts }))
	writeHashes(t, "data/common-leaves.tsv.gz", everybodys(authors, func(a *corpusAuthor) map[string]bool { return a.leaves }))
	deps := everybodys(authors, func(a *corpusAuthor) map[string]bool { return a.deps })
	var rows []string
	for k := range deps {
		rows = append(rows, strings.Replace(k, "\x00", "\t", 1))
	}
	writeRows(t, "data/common-deps.tsv.gz", "# libraries used by three or more people in the corpus: ecosystem, name", rows)
	writeRows(t, "data/usual.tsv.gz", "# how the corpus writes: language, feature, mean rate, spread between people", usualRows(authors))

	// Words: how many projects use each, per language, trimmed the way
	// trimmed() says. The pooled table is added up when the program starts
	// rather than shipped twice.
	tables := map[string]*wordTable{}
	for _, l := range calibrationLanguages {
		tables[l.name] = wordTableOf(authors, l.name)
	}
	rows = nil
	for lang, tb := range tables {
		rows = append(rows, fmt.Sprintf("people\t%s\t%d", lang, tb.people))
		for h, n := range tb.df {
			rows = append(rows, fmt.Sprintf("%s\t%016x\t%d", lang, h, n))
		}
	}
	tables["*"] = pooledTable(tables)
	writeRows(t, "data/words.tsv.gz", "# projects in the corpus using each word: language, word hash, projects", rows)

	// What strangers share: every pair of two different people's projects
	// in a language, kept as two hundred points along the way. Each pair is
	// scored against the corpus with neither person in it, because that is
	// how a project of yours is scored: against a corpus it is not in. The
	// pooled line is scored against the pooled table, the way a language
	// without its own is.
	c := apartFrom(authors)
	rows = nil
	var pooled []float64
	for _, l := range calibrationLanguages {
		mine := withTwo(authors, l.name)
		rows = append(rows, l.name+"\t"+quantiles(c.strangers(mine, l.name, false)))
		pooled = append(pooled, c.strangers(mine, l.name, true)...)
	}
	rows = append(rows, "*\t"+quantiles(pooled))
	writeRows(t, "data/strangers.tsv.gz", "# what pairs of strangers share: language, 201 points from least to most", rows)
}

func quantiles(xs []float64) string {
	sort.Float64s(xs)
	var parts []string
	for i := 0; i <= 200 && len(xs) > 0; i++ {
		parts = append(parts, fmt.Sprintf("%.5g", xs[i*(len(xs)-1)/200]))
	}
	return strings.Join(parts, ",")
}

func writeHashes(t *testing.T, name string, keys map[string]bool) {
	var rows []string
	for h := range hashed(keys) {
		rows = append(rows, fmt.Sprintf("%016x", h))
	}
	writeRows(t, name, "# hashes of what three or more people in the corpus have", rows)
}

func writeRows(t *testing.T, name, header string, rows []string) {
	sort.Strings(rows)
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	z := gzip.NewWriter(f)
	w := bufio.NewWriter(z)
	fmt.Fprintln(w, header)
	for _, r := range rows {
		fmt.Fprintln(w, r)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: %d rows", name, len(rows))
}

// usualRows is each language's average writer: for every feature, the mean
// of each author's rate and how far authors spread around it.
func usualRows(authors []*corpusAuthor) []string {
	byLang := map[string][]map[string]float64{}
	for _, a := range authors {
		if len(a.profiles) == 0 {
			continue
		}
		var pieces []map[string]int
		for _, p := range a.profiles {
			if p != nil {
				pieces = append(pieces, p.Pieces...)
			}
		}
		rates := habitRates(pieces)
		byLang[a.lang] = append(byLang[a.lang], rates)
	}
	var rows []string
	for lang, people := range byLang {
		if len(people) < 10 {
			continue
		}
		seen := map[string]int{}
		for _, p := range people {
			for f := range p {
				seen[f]++
			}
		}
		for f, n := range seen {
			if n*5 < len(people) {
				continue
			}
			mean := 0.0
			for _, p := range people {
				mean += p[f]
			}
			mean /= float64(len(people))
			sd := 0.0
			for _, p := range people {
				sd += (p[f] - mean) * (p[f] - mean)
			}
			sd = math.Sqrt(sd / float64(len(people)))
			rows = append(rows, fmt.Sprintf("%s\t%s\t%.6g\t%.6g", lang, f, mean, sd))
		}
	}
	return rows
}

func spreadOf(xs []float64) string {
	if len(xs) == 0 {
		return "none"
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	q := func(p float64) float64 { return s[int(p*float64(len(s)-1))] }
	return fmt.Sprintf("n=%d min %.2f 10%% %.2f median %.2f 90%% %.2f max %.2f", len(s), s[0], q(.1), q(.5), q(.9), s[len(s)-1])
}

func share(xs []float64, cut float64) float64 {
	n := 0
	for _, x := range xs {
		if x <= cut {
			n++
		}
	}
	return float64(n) / math.Max(1, float64(len(xs)))
}

// separation is the chance a same-author pair sits closer than a
// different-author pair: 0.5 is a coin, 1 is perfect.
func separation(same, different []float64) float64 {
	if len(same) == 0 || len(different) == 0 {
		return math.NaN()
	}
	wins := 0.0
	for _, s := range same {
		for _, d := range different {
			switch {
			case s < d:
				wins++
			case s == d:
				wins += 0.5
			}
		}
	}
	return wins / float64(len(same)*len(different))
}

// wordTableOf counts, over the given authors' projects, how many projects
// use each word. Every project counts, the way the shipped table counts.
func wordTableOf(authors []*corpusAuthor, lang string) *wordTable {
	return trimmed(wordCounts(authors, lang))
}

func wordCounts(authors []*corpusAuthor, lang string) *wordTable {
	t := &wordTable{df: map[uint64]int{}}
	for _, a := range authors {
		if a.lang != lang {
			continue
		}
		for _, w := range a.words {
			t.people++
			for k := range w.Set {
				t.df[hashOf(k)]++
			}
		}
	}
	return t
}

// withTwo is the authors in a language with two projects big enough to
// compare.
func withTwo(authors []*corpusAuthor, lang string) []*corpusAuthor {
	var out []*corpusAuthor
	for _, a := range authors {
		if a.lang == lang && len(a.words) >= 2 {
			out = append(out, a)
		}
	}
	return out
}

// apart is the corpus as somebody outside it sees it. Somebody's own
// projects are never in the tables they are measured against, and the
// strangers line they are set against has to be drawn the same way. Drawn
// against the whole table, every word two strangers shared counted their
// own two projects in how common it was, the line came out too low, and
// strangers measured from outside crossed it three times as often as it
// said: 3 pairs in 100 at the line for 1, and 15 at the line for 5.
type apart struct {
	full   map[string]*wordTable
	pooled *wordTable
}

func apartFrom(authors []*corpusAuthor) *apart {
	c := &apart{full: map[string]*wordTable{}}
	trimmedTables := map[string]*wordTable{}
	for _, l := range calibrationLanguages {
		c.full[l.name] = wordCounts(authors, l.name)
		trimmedTables[l.name] = trimmed(c.full[l.name])
	}
	c.pooled = pooledTable(trimmedTables)
	return c
}

// table is the shipped table for a language, or the pooled one, as it
// would have come out with the given people left out of the corpus. Only
// the words on the given sides are looked up, which is all a comparison
// of those two reads.
func (c *apart) table(lang string, pooled bool, sides []Words, gone ...*corpusAuthor) *wordTable {
	full := c.full[lang]
	t := &wordTable{people: full.people, df: map[uint64]int{}}
	if pooled {
		t.people = c.pooled.people
	}
	for _, p := range gone {
		t.people -= len(p.words)
	}
	trim := func(n int) int {
		if n < fewest {
			return 0
		}
		return n
	}
	for _, side := range sides {
		for k := range side.Set {
			h := hashOf(k)
			if _, done := t.df[h]; done {
				continue
			}
			n := full.df[h]
			for _, p := range gone {
				for _, w := range p.words {
					if w.Set[k] {
						n--
					}
				}
			}
			if pooled {
				// The other languages lose nobody: only this one's share of
				// the pooled count changes.
				t.df[h] = c.pooled.df[h] - trim(full.df[h]) + trim(n)
			} else {
				t.df[h] = trim(n)
			}
		}
	}
	return t
}

// strangers scores every pair of two different people's projects, each
// against the corpus with neither of them in it.
func (c *apart) strangers(people []*corpusAuthor, lang string, pooled bool) []float64 {
	saved := wordTables
	defer func() { wordTables = saved }()
	key := lang
	if pooled {
		key = "*"
	}
	var out []float64
	for i, a := range people {
		for j, b := range people {
			if i == j {
				continue
			}
			x, y := a.words[0], b.words[1]
			wordTables = map[string]*wordTable{key: c.table(lang, pooled, []Words{x, y}, a, b)}
			out = append(out, sharedWords(x, y).Score)
		}
	}
	return out
}

// TestCalibrationWords is the shared rare words measure, judged the honest
// way: rarity is counted on one half of the authors and the pairs come
// from the other half.
func TestCalibrationWords(t *testing.T) {
	authors := readCorpus(t)
	savedTables, savedStrangers := wordTables, strangers
	defer func() { wordTables, strangers = savedTables, savedStrangers }()
	var pooled []float64
	for _, l := range calibrationLanguages {
		var langSame, sameScores, strangerScores []float64
		for split := 0; split < 2; split++ {
			var background, test []*corpusAuthor
			for _, a := range authors {
				switch {
				case a.lang != l.name:
				case halfOf(a.name) == split:
					background = append(background, a)
				case len(a.words) >= 2:
					test = append(test, a)
				}
			}
			wordTables = map[string]*wordTable{l.name: wordTableOf(background, l.name)}
			strangers = map[string][]float64{}
			var these []float64
			for i, a := range test {
				for j, b := range test {
					if i != j {
						these = append(these, sharedWords(a.words[0], b.words[1]).Score)
					}
				}
			}
			sort.Float64s(these)
			strangers[l.name] = these
			strangerScores = append(strangerScores, these...)
			for _, a := range test {
				sh := sharedWords(a.words[0], a.words[1])
				sameScores = append(sameScores, sh.Score)
				langSame = append(langSame, sh.Closer)
			}
		}
		pooled = append(pooled, langSame...)
		neg := func(xs []float64) []float64 {
			out := make([]float64, len(xs))
			for i, x := range xs {
				out[i] = -x
			}
			return out
		}
		t.Logf("%-10s %3d same pairs, separation %.3f, closer than strangers: %s", l.name, len(langSame),
			separation(neg(sameScores), neg(strangerScores)), catchRates(langSame))
	}
	t.Logf("%-10s %3d same pairs, closer than strangers: %s", "all", len(pooled), catchRates(pooled))
}

func catchRates(pcts []float64) string {
	var parts []string
	for _, cut := range []float64{0.8, 0.9, 0.95, 0.98, 0.99} {
		n := 0
		for _, p := range pcts {
			if p >= cut {
				n++
			}
		}
		parts = append(parts, fmt.Sprintf("%.0f%%: %.0f%%", cut*100, 100*float64(n)/math.Max(1, float64(len(pcts)))))
	}
	return strings.Join(parts, ", ")
}

// TestCalibrationShipped measures the tables that ship rather than tables
// built for the test: each person's two projects against the corpus
// without them, and pairs of strangers against the corpus without either,
// both set against the strangers line in data/.
func TestCalibrationShipped(t *testing.T) {
	authors := readCorpus(t)
	c := apartFrom(authors)
	saved := wordTables
	defer func() { wordTables = saved }()
	var same, different []float64
	for _, l := range calibrationLanguages {
		mine := withTwo(authors, l.name)
		var langSame []float64
		for i, a := range mine {
			x := a.words[0]
			wordTables = map[string]*wordTable{l.name: c.table(l.name, false, []Words{x, a.words[1]}, a)}
			langSame = append(langSame, sharedWords(x, a.words[1]).Closer)
			for j := 1; j <= 3; j++ {
				b := mine[(i+j)%len(mine)]
				if b == a {
					continue
				}
				wordTables = map[string]*wordTable{l.name: c.table(l.name, false, []Words{x, b.words[1]}, a, b)}
				different = append(different, sharedWords(x, b.words[1]).Closer)
			}
		}
		same = append(same, langSame...)
		t.Logf("%-10s %3d same pairs, closer than strangers: %s", l.name, len(langSame), catchRates(langSame))
	}
	t.Logf("%-10s %3d same pairs, closer than strangers: %s", "all", len(same), catchRates(same))
	t.Logf("%-10s %3d pairs, closer than strangers: %s", "strangers", len(different), catchRates(different))
}

// TestCalibrationTied is the question the exit code answers: how often two
// projects come out tied together, on any count, when one person wrote
// both and when two people did.
func TestCalibrationTied(t *testing.T) {
	authors := readCorpus(t)
	saved := [3]map[uint64]bool{commonTexts, commonLeaves, commonDeps}
	savedTables, savedStrangers := wordTables, strangers
	defer func() {
		commonTexts, commonLeaves, commonDeps = saved[0], saved[1], saved[2]
		wordTables, strangers = savedTables, savedStrangers
	}()
	type tally struct{ pairs, names, text, words, any, unnamed int }
	var same, different tally
	for split := 0; split < 2; split++ {
		var background, test []*corpusAuthor
		for _, a := range authors {
			if halfOf(a.name) == split {
				background = append(background, a)
			} else if len(a.sources) >= 2 && len(a.words) >= 2 {
				test = append(test, a)
			}
		}
		commonTexts = hashed(everybodys(background, func(a *corpusAuthor) map[string]bool { return a.texts }))
		commonLeaves = hashed(everybodys(background, func(a *corpusAuthor) map[string]bool { return a.leaves }))
		commonDeps = hashed(everybodys(background, func(a *corpusAuthor) map[string]bool { return a.deps }))
		wordTables, strangers = map[string]*wordTable{}, map[string][]float64{}
		for _, l := range calibrationLanguages {
			wordTables[l.name] = wordTableOf(background, l.name)
			var these []float64
			for _, a := range test {
				for _, b := range test {
					if a != b && a.lang == l.name && b.lang == l.name {
						these = append(these, sharedWords(a.words[0], b.words[1]).Score)
					}
				}
			}
			sort.Float64s(these)
			strangers[l.name] = these
		}
		count := func(a, b *corpusAuthor, into *tally, x, y int) {
			into.pairs++
			named := len(sharedLeavesOf(distinctLeaves(leavesOf(a.sources[x])), distinctLeaves(leavesOf(b.sources[y])))) > 0
			text := false
			for _, f := range sharedFragments(a.sources[x], b.sources[y]) {
				if f.Kind != "lines" || f.Lines >= blockLines {
					text = true
				}
			}
			words := sharedWords(a.words[x], b.words[y]).Closer >= wordsSame
			for _, c := range []struct {
				yes bool
				n   *int
			}{{named, &into.names}, {text, &into.text}, {words, &into.words}, {named || text || words, &into.any}, {text || words, &into.unnamed}} {
				if c.yes {
					*c.n++
				}
			}
		}
		for i, a := range test {
			count(a, a, &same, 0, 1)
			for j := 1; j <= 3; j++ {
				b := test[(i+j)%len(test)]
				if b != a && b.lang == a.lang {
					count(a, b, &different, 0, 1)
				}
			}
		}
	}
	show := func(name string, x tally) {
		pc := func(n int) string { return fmt.Sprintf("%5.1f%%", 100*float64(n)/math.Max(1, float64(x.pairs))) }
		t.Logf("%-9s %4d pairs tied: by names %s, by shared text %s, by words %s, by any %s, by any but names %s",
			name, x.pairs, pc(x.names), pc(x.text), pc(x.words), pc(x.any), pc(x.unnamed))
	}
	show("same", same)
	show("different", different)
}
