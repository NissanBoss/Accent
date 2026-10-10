package main

// Taking code apart into the pieces a style is made of.
//
// This is not a parser and does not try to be one. It knows where comments
// and strings begin and end, which words belong to the language, and what
// sits on either side of every token, because that is where habits live:
// whether somebody writes "if (" or "if(", "x = 1" or "x=1", a comment with
// a capital letter and a full stop or without. A parser would throw most of
// that away before it got to the interesting part.

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Kind byte

const (
	Ident Kind = iota
	Keyword
	Number
	Str
	Comment
	Punct
)

type Token struct {
	Kind Kind
	Text string
	Line int
	// Before and After are what touches the token on its line: ' ' for
	// whitespace, '\n' for the start or the end of the line, 0 for another
	// token right against it.
	Before, After byte
	// Own is set on a comment that has its line to itself, rather than
	// following code.
	Own bool
	// Doc is set on a Python string that stands alone on its line, which is
	// a docstring and is read as a comment.
	Doc bool
}

// Layout is how the lines of a file sit on the page.
type Layout struct {
	Lines, Blank, Trailing     int
	TabIndented, SpaceIndented int
	// Steps counts how far each indentation went in from the line above,
	// in spaces: mostly 2 or 4, and that is the habit.
	Steps           map[int]int
	Over80, Over100 int
	Length          int
	CRLF            int
	// Doubled is how often two blank lines come together.
	Doubled int
}

var operators = []string{
	"...", "===", "!==", "**=", "<<=", ">>=", ">>>", "->", "=>", "::", ":=", "==", "!=", "<=", ">=",
	"&&", "||", "++", "--", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<", ">>", "..", "?.",
	"??", "**", "<-", "//",
}

func lex(lang *Language, src string) ([]Token, Layout) {
	var out []Token
	line := 1
	codeOnLine := false
	i := 0
	n := len(src)

	before := func(at int) byte {
		if at == 0 || src[at-1] == '\n' {
			return '\n'
		}
		j := at - 1
		for j >= 0 && (src[j] == ' ' || src[j] == '\t') {
			j--
		}
		switch {
		case j < 0 || src[j] == '\n':
			return '\n'
		case j < at-1:
			return ' '
		}
		return 0
	}
	after := func(at int) byte {
		switch {
		case at >= n || src[at] == '\n' || src[at] == '\r':
			return '\n'
		case src[at] == ' ' || src[at] == '\t':
			j := at
			for j < n && (src[j] == ' ' || src[j] == '\t') {
				j++
			}
			if j >= n || src[j] == '\n' || src[j] == '\r' {
				return '\n'
			}
			return ' '
		}
		return 0
	}
	emit := func(kind Kind, from, to int) {
		t := Token{Kind: kind, Text: src[from:to], Line: line, Before: before(from), After: after(to)}
		if kind == Comment {
			t.Own = !codeOnLine
		} else {
			codeOnLine = true
		}
		out = append(out, t)
		line += strings.Count(src[from:to], "\n")
	}

	for i < n {
		c := src[i]
		switch {
		case c == '\n':
			line++
			codeOnLine = false
			i++
			continue
		case c == ' ' || c == '\t' || c == '\r' || c == '\f':
			i++
			continue
		}

		if end, ok := commentAt(lang, src, i, codeOnLine); ok {
			emit(Comment, i, end)
			i = end
			continue
		}
		if end, raw, ok := stringAt(lang, src, i); ok {
			doc := lang.Name == "Python" && raw && !codeOnLine
			emit(Str, i, end)
			out[len(out)-1].Doc = doc
			i = end
			continue
		}

		switch {
		case c >= '0' && c <= '9' || c == '.' && i+1 < n && src[i+1] >= '0' && src[i+1] <= '9':
			j := i + 1
			hex := c == '0' && j < n && (src[j] == 'x' || src[j] == 'X')
			for j < n {
				d := src[j]
				switch {
				case d == '.' && j+1 < n && src[j+1] == '.':
				case isWordByte(d) || d == '.':
					j++
					continue
				case (d == '+' || d == '-') && !hex && (src[j-1] == 'e' || src[j-1] == 'E'):
					j++
					continue
				}
				break
			}
			emit(Number, i, j)
			i = j
		case isWordStart(src, i):
			j := i
			for j < n && isWordAt(src, j) {
				_, size := utf8.DecodeRuneInString(src[j:])
				j += size
			}
			// Ruby's empty? and save! are one name each.
			if lang.Name == "Ruby" && j < n && (src[j] == '?' || src[j] == '!') && (j+1 >= n || src[j+1] != '=') {
				j++
			}
			kind := Ident
			if lang.Keywords[src[i:j]] {
				kind = Keyword
			}
			emit(kind, i, j)
			i = j
		default:
			width := 1
			for _, op := range operators {
				if strings.HasPrefix(src[i:], op) {
					width = len(op)
					break
				}
			}
			if c >= utf8.RuneSelf {
				_, width = utf8.DecodeRuneInString(src[i:])
			}
			emit(Punct, i, i+width)
			i += width
		}
	}
	return out, layoutOf(src)
}

func isWordByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isWordStart(src string, i int) bool {
	c := src[i]
	if c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= utf8.RuneSelf {
		r, _ := utf8.DecodeRuneInString(src[i:])
		return unicode.IsLetter(r)
	}
	return false
}

func isWordAt(src string, i int) bool {
	c := src[i]
	if c < utf8.RuneSelf {
		return isWordByte(c)
	}
	r, _ := utf8.DecodeRuneInString(src[i:])
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// commentAt says whether a comment starts here, and where it ends.
func commentAt(lang *Language, src string, i int, codeOnLine bool) (int, bool) {
	for _, b := range lang.Blocks {
		if !strings.HasPrefix(src[i:], b[0]) {
			continue
		}
		depth, j := 1, i+len(b[0])
		for j < len(src) && depth > 0 {
			switch {
			case strings.HasPrefix(src[j:], b[1]):
				depth--
				j += len(b[1])
			case lang.Nested && strings.HasPrefix(src[j:], b[0]):
				depth++
				j += len(b[0])
			default:
				j++
			}
		}
		return j, true
	}
	starts := lang.Line
	if lang.Hash && src[i] == '#' {
		// In the shell a # is a comment only at the start of a word: in
		// ${#list} or $# it is part of the expression.
		if lang.Name == "Shell" && i > 0 && src[i-1] != ' ' && src[i-1] != '\t' && src[i-1] != '\n' {
			return 0, false
		}
		// PHP 8 writes attributes as #[...].
		if lang.Name == "PHP" && strings.HasPrefix(src[i:], "#[") {
			return 0, false
		}
		starts = append(starts, "#")
	}
	for _, s := range starts {
		if strings.HasPrefix(src[i:], s) {
			j := strings.IndexByte(src[i:], '\n')
			if j < 0 {
				return len(src), true
			}
			end := i + j
			if end > i && src[end-1] == '\r' {
				end--
			}
			return end, true
		}
	}
	return 0, false
}

// stringAt says whether a string starts here, where it ends, and whether it
// was a raw one, which for Python means three quotes.
func stringAt(lang *Language, src string, i int) (int, bool, bool) {
	// Python's prefixes, f"" and rb"" and the rest, belong to the string.
	if lang.Name == "Python" {
		j := i
		for j < len(src) && j-i < 2 && strings.IndexByte("rRbBfFuU", src[j]) >= 0 {
			j++
		}
		if j > i && j < len(src) && (src[j] == '"' || src[j] == '\'') && (i == 0 || !isWordByte(src[i-1])) {
			i = j
		}
	}
	if lang.Name == "Rust" && src[i] == 'r' && i+1 < len(src) && (src[i+1] == '"' || src[i+1] == '#') &&
		(i == 0 || !isWordByte(src[i-1])) {
		j := i + 1
		hashes := 0
		for j < len(src) && src[j] == '#' {
			hashes++
			j++
		}
		if j < len(src) && src[j] == '"' {
			closing := "\"" + strings.Repeat("#", hashes)
			if k := strings.Index(src[j+1:], closing); k >= 0 {
				return j + 1 + k + len(closing), true, true
			}
			return len(src), true, true
		}
	}
	for _, r := range lang.Raw {
		if strings.HasPrefix(src[i:], r) {
			if k := strings.Index(src[i+len(r):], r); k >= 0 {
				return i + len(r) + k + len(r), len(r) == 3, true
			}
			return len(src), len(r) == 3, true
		}
	}
	q := src[i]
	if strings.IndexByte(lang.Quotes, q) < 0 {
		return 0, false, false
	}
	if q == '\'' && lang.Name == "Rust" {
		// 'a is a lifetime and 'a' is a character. Only the second is a
		// string.
		if i+2 < len(src) && src[i+1] != '\\' && src[i+2] != '\'' {
			return 0, false, false
		}
	}
	j := i + 1
	for j < len(src) {
		switch src[j] {
		case '\\':
			j += 2
			continue
		case q:
			return j + 1, false, true
		case '\n':
			// Only the shell, Ruby and PHP let a quoted string run on to
			// the next line. Anywhere else a quote left open is a typo or
			// an apostrophe, and running on would swallow the file.
			if lang.Name != "Shell" && lang.Name != "Ruby" && lang.Name != "PHP" {
				return j, false, true
			}
		}
		j++
	}
	return len(src), false, true
}

func layoutOf(src string) Layout {
	l := Layout{Steps: map[int]int{}}
	lines := strings.Split(src, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lastIndent, blankRun := -1, 0
	for _, raw := range lines {
		if strings.HasSuffix(raw, "\r") {
			l.CRLF++
			raw = strings.TrimSuffix(raw, "\r")
		}
		l.Lines++
		trimmed := strings.TrimRight(raw, " \t")
		if trimmed == "" {
			l.Blank++
			blankRun++
			if blankRun == 2 {
				l.Doubled++
			}
			continue
		}
		blankRun = 0
		if len(trimmed) < len(raw) {
			l.Trailing++
		}
		width := utf8.RuneCountInString(strings.ReplaceAll(raw, "\t", "    "))
		l.Length += width
		if width > 80 {
			l.Over80++
		}
		if width > 100 {
			l.Over100++
		}
		indent := len(trimmed) - len(strings.TrimLeft(trimmed, " \t"))
		switch {
		case indent == 0:
		case trimmed[0] == '\t':
			l.TabIndented++
		default:
			l.SpaceIndented++
		}
		if trimmed[0] != '\t' && lastIndent >= 0 && indent > lastIndent && indent-lastIndent <= 8 {
			l.Steps[indent-lastIndent]++
		}
		lastIndent = indent
	}
	return l
}
