package main

import (
	"strings"
	"testing"
)

func kinds(tokens []Token) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString("IKNSCP"[t.Kind : t.Kind+1])
	}
	return b.String()
}

func TestTheLexerKnowsCommentsAndStrings(t *testing.T) {
	cases := []struct {
		lang, src, want string
	}{
		{"Go", "x := `raw // not a comment` // a comment\n", "IPSC"},
		{"Python", "s = f\"{x}\" # note\n\"\"\"doc\"\"\"\n", "IPSCS"},
		{"Rust", "fn f<'a>(x: &'a str) -> char { 'c' } /* a /* nested */ comment */", "KIPPIPPIPPPIIPPIPSPC"},
		{"Shell", "echo ${#list} # count\n", "KIPPIPC"},
		{"JavaScript", "let s = `a ${b} c`; // x\n", "KIPSPC"},
	}
	for _, c := range cases {
		var lang *Language
		for _, l := range languages {
			if l.Name == c.lang {
				lang = l
			}
		}
		tokens, _ := lex(lang, c.src)
		if got := kinds(tokens); got != c.want {
			var texts []string
			for _, tk := range tokens {
				texts = append(texts, tk.Text)
			}
			t.Errorf("%s %q: got %s want %s (%q)", c.lang, c.src, got, c.want, texts)
		}
	}
}
