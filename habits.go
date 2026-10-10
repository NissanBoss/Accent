package main

// What sets one project's style apart, in words.
//
// Two projects are compared on their words, in vocabulary.go. Habits are
// what Accent says about one project on its own: which features this code
// uses far more or far less than the people in the calibration corpus who
// write the same language, said in sentences a person recognises. The
// comments without a capital letter, every error called e, the space
// nobody else puts before a bracket.

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// layoutFeature says whether a feature is spacing, which a formatter
// decides for everybody who runs it.
func layoutFeature(f string) bool {
	return strings.HasPrefix(f, "p:") || f == "c://_" || f == "c:#_"
}

// describe turns a feature into a few words a person recognises.
func describe(f string) string {
	kind, what, _ := strings.Cut(f, ":")
	switch kind {
	case "k":
		return "`" + what + "`"
	case "i":
		return "the name `" + what + "`"
	case "n":
		switch what {
		case "snake":
			return "snake_case names"
		case "camel":
			return "camelCase names"
		case "Pascal":
			return "PascalCase names"
		case "UPPER":
			return "names in capitals"
		case "lower":
			return "names of one lower case word"
		case "Mixed_case":
			return "names mixing capitals and underscores"
		case "len4-7":
			return "names of four to seven letters"
		case "len8-12":
			return "names of eight to twelve letters"
		case "len13-20":
			return "long names, thirteen to twenty letters"
		case "len21+":
			return "very long names, over twenty letters"
		}
	case "#":
		switch what {
		case "hex":
			return "numbers in hex"
		case "float":
			return "numbers with a decimal point"
		case "digit":
			return "single digit numbers other than 0 and 1"
		case "int":
			return "numbers of two digits or more"
		}
		return "the number " + what
	case "s":
		return describeQuote(what)
	case "c":
		return describeComment(what)
	case "w":
		return fmt.Sprintf("%q in comments", what)
	case "p":
		return describePunct(what)
	case "much":
		if kind, ok := muchOf[what]; ok {
			return kind + ", for the amount of code"
		}
	}
	return f
}

var muchOf = map[string]string{
	"c": "comments", "w": "words in comments", "i": "short names", "n": "long names",
	"s": "strings", "#": "numbers", "k": "keywords", "p": "punctuation",
}

func describeQuote(q string) string {
	empty := strings.HasSuffix(q, "empty")
	q = strings.TrimSuffix(q, "empty")
	name := map[string]string{
		`"`: "double quoted strings", `'`: "single quoted strings", "`": "strings in backticks",
		`"""`: "triple quoted strings", `'''`: "triple single quoted strings",
		`f"`: "f-strings", `f'`: "f-strings in single quotes", `r"`: "raw strings", `b"`: "byte strings",
	}[q]
	if name == "" {
		name = "strings opened with " + q
	}
	if empty {
		return "empty " + name
	}
	return name
}

func describeComment(c string) string {
	switch c {
	case "own":
		return "comments on a line of their own"
	case "after-code":
		return "comments at the end of a line of code"
	case "Capital":
		return "comments that start with a capital letter"
	case "lower":
		return "comments that start in lower case"
	case "full-stop":
		return "comments that end with a full stop"
	}
	if marker, ok := strings.CutSuffix(c, "_"); ok {
		return "a space after " + marker + " in comments"
	}
	if c == `"""` || c == "'''" {
		return "docstrings"
	}
	return "comments opened with " + c
}

func describePunct(p string) string {
	parts := strings.Fields(p)
	if len(parts) != 3 {
		return "`" + p + "`"
	}
	before := map[string]string{"space": "a space before it", "edge": "starting a line", "none": "nothing before it"}[parts[0]]
	after := map[string]string{"space": "a space after", "edge": "ending the line", "none": "nothing after"}[parts[2]]
	return fmt.Sprintf("`%s` with %s and %s", parts[1], before, after)
}

// unusual lists what sets one profile apart from the people in the
// calibration corpus who write the same language, furthest first. It needs
// the corpus table, and says nothing about features that table lacks.
func unusual(p *Profile, howMany int) []string {
	base := usual[p.Lang]
	if len(base) == 0 {
		return nil
	}
	rates := habitRates(p.Pieces)
	if len(rates) == 0 {
		return nil
	}
	type odd struct {
		feature string
		z, x    float64
	}
	var odds []odd
	formatter := languageNamed(p.Lang).Formatter
	for f, b := range base {
		if b[1] == 0 || b[0] == 0 || formatter != "" && layoutFeature(f) {
			continue
		}
		rate := rates[f]
		if z := (rate - b[0]) / b[1]; math.Abs(z) >= 2 {
			odds = append(odds, odd{f, z, rate / b[0]})
		}
	}
	sort.Slice(odds, func(i, j int) bool {
		if math.Abs(odds[i].z) != math.Abs(odds[j].z) {
			return math.Abs(odds[i].z) > math.Abs(odds[j].z)
		}
		return odds[i].feature < odds[j].feature
	})
	var out []string
	for _, o := range odds[:min(howMany, len(odds))] {
		out = append(out, describe(o.feature)+", "+howOften(o.x))
	}
	return out
}

func howOften(x float64) string {
	switch {
	case x == 0:
		return "never, where most people do"
	case x >= 10:
		return "more than ten times as often as most people"
	case x >= 1.5:
		return fmt.Sprintf("%.0f times as often as most people", math.Round(x))
	case x <= 0.1:
		return "almost never, where most people do"
	case x <= 0.67:
		return fractionWord(x) + " as often as most people"
	}
	return "more than most people"
}

func fractionWord(x float64) string {
	switch {
	case x < 0.2:
		return "a tenth"
	case x < 0.3:
		return "a quarter"
	case x < 0.4:
		return "a third"
	}
	return "half"
}

// habitRates says how often each feature is used within its own kind: a
// word in comments as a share of the words in comments, a name as a share
// of names. Measured against everything instead, somebody who writes long
// comments came out using every small word ten times as often as anybody,
// which says one thing ten times. How much of each kind there is goes in
// as a feature of its own, so the long comments are still said, once.
func habitRates(pieces []map[string]int) map[string]float64 {
	counts := map[string]int{}
	kinds := map[string]int{}
	total := 0
	for _, piece := range pieces {
		for f, n := range piece {
			counts[f] += n
			kind, _, _ := strings.Cut(f, ":")
			kinds[kind] += n
			total += n
		}
	}
	rates := map[string]float64{}
	if total == 0 {
		return rates
	}
	for f, n := range counts {
		kind, _, _ := strings.Cut(f, ":")
		rates[f] = float64(n) / float64(kinds[kind])
	}
	for kind, n := range kinds {
		rates["much:"+kind] = float64(n) / float64(total)
	}
	return rates
}
