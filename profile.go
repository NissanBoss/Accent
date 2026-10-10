package main

// Turning code into the counts a style is compared on.
//
// Every token becomes a feature that says something about how it was
// written rather than what it is about. A keyword stays itself. A long name
// becomes its shape, snake or camel case and how long, because the name
// itself is the project's vocabulary and two projects about invoices would
// otherwise look like one author. Short names stay as they are, because i,
// err, e, tmp and res are exactly the choices that differ between people.
// Punctuation keeps what touches it, so "if (" and "if(" are different
// features. Comments keep their shape, and their small words.
//
// Then the stream is cut into pieces of the same length, so a long file and
// a short one count the same, and the counts are set against how the
// people in the calibration corpus write the same language, in habits.go.

import (
	"strings"
	"unicode"
)

// pieceSize is how many features make a piece.
const pieceSize = 1000

type Profile struct {
	Lang   string
	Pieces []map[string]int
	Files  int
	Tokens int
}

// habitNames are names longer than three letters that are still a choice
// between people rather than part of what the project is about.
var habitNames = wordSet("self this that error errors index count value values result results " +
	"temp item items elem element data args kwargs opts options config conf params param argv " +
	"callback next prev curr current first last left right start stop begin done found total length " +
	"size list dict array output input buffer line lines text name path file files node nodes event " +
	"cancel resp response request reply client server conn logger retval")

// smallWords are the words in comments that say nothing about the subject,
// which is why they say something about the writer.
var smallWords = wordSet("a about above after again against all also am an and any are as at be because " +
	"been before being below between both but by can could did do does doing down during each else " +
	"ever every few for from further had has have having he her here hers him his how however i if " +
	"in into is it its itself just least less let like many may maybe me might more most much must " +
	"my never no nor not now of off often on once one only or other otherwise our out over own per " +
	"perhaps quite rather same she should since so some still such than that the their them then " +
	"there therefore these they this those though through thus to too under until up upon us very " +
	"was we were what when where whether which while who whom whose why will with within without " +
	"would yet you your")

func profilesOf(s *Source) map[string]*Profile {
	byLang := map[string]*Profile{}
	streams := map[string][]string{}
	for _, f := range s.Code {
		p := byLang[f.Lang.Name]
		if p == nil {
			p = &Profile{Lang: f.Lang.Name}
			byLang[f.Lang.Name] = p
		}
		p.Files++
		tokens, _ := lex(f.Lang, f.Text)
		p.Tokens += len(tokens)
		streams[f.Lang.Name] = append(streams[f.Lang.Name], features(tokens)...)
	}
	for lang, stream := range streams {
		p := byLang[lang]
		for start := 0; start+pieceSize <= len(stream); start += pieceSize {
			counts := map[string]int{}
			for _, f := range stream[start : start+pieceSize] {
				counts[f]++
			}
			p.Pieces = append(p.Pieces, counts)
		}
	}
	return byLang
}

func features(tokens []Token) []string {
	var out []string
	for _, t := range tokens {
		switch t.Kind {
		case Keyword:
			out = append(out, "k:"+t.Text)
		case Ident:
			out = append(out, nameFeatures(t.Text)...)
		case Number:
			out = append(out, "#:"+numberShape(t.Text))
		case Str:
			if t.Doc {
				out = append(out, commentFeatures(t)...)
				continue
			}
			out = append(out, "s:"+quoteShape(t.Text))
		case Comment:
			out = append(out, commentFeatures(t)...)
		case Punct:
			out = append(out, "p:"+side(t.Before)+" "+t.Text+" "+side(t.After))
		}
	}
	return out
}

// side names what touches a token: a space, the edge of the line, or
// nothing. The punctuation goes between, with spaces around it, because no
// punctuation contains a space and plenty contains | and _.
func side(b byte) string {
	switch b {
	case ' ':
		return "space"
	case '\n':
		return "edge"
	}
	return "none"
}

func nameFeatures(name string) []string {
	if len(name) <= 3 || habitNames[name] {
		return []string{"i:" + name}
	}
	return []string{"n:" + nameCase(name), "n:" + lengthBand(len(name))}
}

func nameCase(name string) string {
	hasUpper, hasLower, hasUnder := false, false, strings.Contains(name, "_")
	for _, r := range name {
		hasUpper = hasUpper || unicode.IsUpper(r)
		hasLower = hasLower || unicode.IsLower(r)
	}
	first := []rune(name)[0]
	switch {
	case hasUpper && !hasLower:
		return "UPPER"
	case hasUnder && hasLower && !hasUpper:
		return "snake"
	case hasUnder:
		return "Mixed_case"
	case unicode.IsUpper(first):
		return "Pascal"
	case hasUpper:
		return "camel"
	}
	return "lower"
}

func lengthBand(n int) string {
	switch {
	case n <= 7:
		return "len4-7"
	case n <= 12:
		return "len8-12"
	case n <= 20:
		return "len13-20"
	}
	return "len21+"
}

func numberShape(s string) string {
	switch {
	case s == "0" || s == "1":
		return s
	case strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X"):
		return "hex"
	case strings.ContainsAny(s, ".eE") && !strings.HasPrefix(s, "0x"):
		return "float"
	case len(s) == 1:
		return "digit"
	}
	return "int"
}

func quoteShape(s string) string {
	for i, c := range s {
		if c == '"' || c == '\'' || c == '`' {
			q := s[:i+1]
			if strings.HasPrefix(s[i:], `"""`) || strings.HasPrefix(s[i:], "'''") {
				q = s[:i+3]
			}
			if len(s)-len(q) <= len(q) {
				return q + "empty"
			}
			return q
		}
	}
	return "other"
}

// commentFeatures turns a comment into its shape: how it is opened, whether
// a space follows, whether it starts with a capital and ends with a full
// stop, whether it has a line to itself, and its small words.
func commentFeatures(t Token) []string {
	text := t.Text
	marker := ""
	for _, m := range []string{"///", "//!", "//", "/**", "/*", "#!", "#", "--", `"""`, "'''"} {
		if strings.HasPrefix(text, m) {
			marker = m
			break
		}
	}
	body := strings.TrimPrefix(text, marker)
	out := []string{"c:" + marker}
	if body != "" && (body[0] == ' ' || body[0] == '\t') {
		out = append(out, "c:"+marker+"_")
	}
	body = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(body, "*/"), `"""`))
	if t.Own {
		out = append(out, "c:own")
	} else {
		out = append(out, "c:after-code")
	}
	letters := []rune(body)
	if len(letters) > 0 && unicode.IsLetter(letters[0]) {
		if unicode.IsUpper(letters[0]) {
			out = append(out, "c:Capital")
		} else {
			out = append(out, "c:lower")
		}
	}
	if strings.HasSuffix(body, ".") {
		out = append(out, "c:full-stop")
	}
	for _, w := range strings.FieldsFunc(strings.ToLower(body), func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	}) {
		if smallWords[w] {
			out = append(out, "w:"+w)
		}
	}
	return out
}
