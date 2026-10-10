package main

// The words two projects share that few people use.
//
// This is where the calibration ended up, and not where it started. The
// first measure was Burrows's Delta over how often each keyword, bracket and
// kind of name turns up, Tell's method moved from prose to code, and on real
// people's repositories it barely told the same person on two projects from
// two strangers: what a project is about moves those counts far more than
// who wrote it. Scaling every feature by how much people differ helped a
// little. What worked was the opposite of counting: the rare words. The
// verbs somebody reaches for in names, load or read or fetch, the
// abbreviations, cfg or conf, the words in their comments and messages,
// and whole names they reuse. One person's projects share words that few
// other people use; strangers rarely do.
//
// Rare means used in no more than one project in ten in the calibration
// corpus, and each shared word counts by how rare it is.

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// rareShare is how common a word may be and still count: in at most this
// share of the projects in the corpus.
const rareShare = 0.1

// minTokens is the least code in one language the comparison is made on.
// Below it there are too few words for sharing them to mean anything.
const minTokens = 3000

type Words struct {
	Lang   string
	Tokens int
	Set    map[string]bool
}

// wordsOf collects, for one language in a source, the words of its names,
// comments and messages, as kind:word.
func wordsOf(s *Source, lang string) Words {
	w := Words{Lang: lang, Set: map[string]bool{}}
	add := func(kind, word string) {
		if word != "" {
			w.Set[kind+":"+word] = true
		}
	}
	for _, f := range s.Code {
		if f.Lang.Name != lang {
			continue
		}
		tokens, _ := lex(f.Lang, f.Text)
		w.Tokens += len(tokens)
		for _, t := range tokens {
			switch t.Kind {
			case Ident:
				if len(t.Text) <= 3 {
					add("short", t.Text)
				} else {
					add("name", t.Text)
				}
				parts := subwords(t.Text)
				for i, p := range parts {
					add("part", p)
					if i > 0 {
						add("parts", parts[i-1]+" "+p)
					}
				}
			case Comment:
				said := commentWords(t.Text)
				for i, word := range said {
					add("said", word)
					if i > 0 {
						add("saids", said[i-1]+" "+word)
					}
				}
			case Str:
				body := strings.Trim(t.Text, "\"'`")
				if ws := strings.Fields(strings.ToLower(body)); len(ws) >= 2 {
					for _, word := range ws {
						if word = strings.Trim(word, ":,.%!?()-"); letters(word) >= 3 {
							add("message", word)
						}
					}
				}
			}
		}
	}
	return w
}

// subwords splits a name into the words it is made of: parseRetryAfter
// into parse, retry and after, MAX_RETRIES into max and retries.
func subwords(name string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) >= 2 {
			out = append(out, strings.ToLower(string(cur)))
		}
		cur = cur[:0]
	}
	rs := []rune(name)
	for i, r := range rs {
		switch {
		case r == '_' || r == '$' || unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r) && len(cur) > 0 &&
			(unicode.IsLower(cur[len(cur)-1]) || i+1 < len(rs) && unicode.IsLower(rs[i+1])):
			flush()
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return out
}

func commentWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(proseOf(text)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r == '\'' || r >= 0x80 && unicode.IsLetter(r))
	})
}

// rarity is how much a word counts: nothing if more than one project in
// ten in the corpus uses it, and more the fewer do.
func rarity(lang, key string) float64 {
	t := wordTables[lang]
	if t == nil {
		t = wordTables["*"]
	}
	if t == nil || t.people == 0 {
		return 1
	}
	share := float64(t.df[hashOf(key)]+1) / float64(t.people+1)
	if share > rareShare {
		return 0
	}
	return math.Log(1 / share)
}

type Shared struct {
	Lang  string
	Score float64
	// Closer is the share of pairs of strangers in the corpus that are
	// further apart than these two, which is the number the report gives.
	Closer float64
	Ready  bool
	// Words are the shared words, rarest first, for saying what did it.
	Words []string
}

// sharedWords measures how much of two projects' rare vocabulary is the
// same: twice the weight of the shared words over the weight of all of
// them, which is 0 for nothing in common and 1 for the same words.
func sharedWords(a, b Words) Shared {
	sh := Shared{Lang: a.Lang}
	if a.Tokens < minTokens || b.Tokens < minTokens {
		return sh
	}
	type weighed struct {
		key string
		w   float64
	}
	var both []weighed
	total, shared := 0.0, 0.0
	for k := range a.Set {
		w := rarity(a.Lang, k)
		total += w
		if b.Set[k] && w > 0 {
			shared += w
			both = append(both, weighed{k, w})
		}
	}
	for k := range b.Set {
		total += rarity(a.Lang, k)
	}
	if total == 0 {
		return sh
	}
	sh.Score = 2 * shared / total
	sh.Ready = true
	sh.Closer = closerThan(a.Lang, sh.Score)
	// Rarest first, and among the equally rare the ones a person would
	// recognise as theirs: a whole name before a part of one, a phrase
	// before a word, a long word before a short one.
	sort.Slice(both, func(i, j int) bool {
		if both[i].w != both[j].w {
			return both[i].w > both[j].w
		}
		ki, kj := kindOrder(both[i].key), kindOrder(both[j].key)
		if ki != kj {
			return ki < kj
		}
		if len(both[i].key) != len(both[j].key) {
			return len(both[i].key) > len(both[j].key)
		}
		return both[i].key < both[j].key
	})
	for _, x := range both {
		sh.Words = append(sh.Words, x.key)
	}
	return sh
}

// closerThan is the share of stranger pairs in the corpus that scored
// lower than this.
func closerThan(lang string, score float64) float64 {
	qs := strangers[lang]
	if len(qs) == 0 {
		qs = strangers["*"]
	}
	if len(qs) == 0 {
		return 0
	}
	return float64(sort.SearchFloat64s(qs, score)) / float64(len(qs))
}

// fewest is the least projects a word must be in for the table to keep it.
// A word in one project or two reads the same as a word in none, and
// leaving them out keeps the table to a size worth shipping.
const fewest = 3

func trimmed(t *wordTable) *wordTable {
	out := &wordTable{people: t.people, df: map[uint64]int{}}
	for h, n := range t.df {
		if n >= fewest {
			out.df[h] = n
		}
	}
	return out
}

// pooledTable is every language's table added together, which is what a
// language with no corpus of its own is measured against.
func pooledTable(tables map[string]*wordTable) *wordTable {
	out := &wordTable{df: map[uint64]int{}}
	langs := make([]string, 0, len(tables))
	for lang := range tables {
		if lang != "*" {
			langs = append(langs, lang)
		}
	}
	sort.Strings(langs)
	for _, lang := range langs {
		out.people += tables[lang].people
		for h, n := range tables[lang].df {
			out.df[h] += n
		}
	}
	return out
}

func kindOrder(key string) int {
	kind, _, _ := strings.Cut(key, ":")
	for i, k := range []string{"name", "saids", "parts", "said", "message", "part", "short"} {
		if k == kind {
			return i
		}
	}
	return 99
}

func letters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}
