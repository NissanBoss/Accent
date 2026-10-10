package main

// Keeping the report safe to paste somewhere.
//
// The report is about what ties somebody's two names together, so the
// names cannot be in it: an email address, a GitHub account and the line
// both projects share are masked down to their first and last letters
// unless --reveal is given. That is enough for the owner to recognise their
// own and not enough for anybody else to search for. Where a line is, file
// and number, is always shown, because it is what the owner needs in order
// to go and change it.

import "strings"

// maskWord keeps the first and last letter.
func maskWord(s string) string {
	r := []rune(s)
	switch {
	case len(r) == 0:
		return ""
	case len(r) <= 2:
		return string(r[0]) + strings.Repeat("*", len(r)-1)
	}
	return string(r[0]) + strings.Repeat("*", len(r)-2) + string(r[len(r)-1])
}

// maskText masks every word in a line and keeps the punctuation, so code
// keeps its shape: f*****n l******g(c**g *C****g). Secrets were covered
// before this and stay covered.
func maskText(s string) string {
	parts := strings.Split(s, covered)
	for i, p := range parts {
		var b strings.Builder
		var word []rune
		flush := func() {
			if len(word) > 0 {
				b.WriteString(maskWord(string(word)))
				word = word[:0]
			}
		}
		for _, r := range p {
			if isWordRune(r) {
				word = append(word, r)
				continue
			}
			flush()
			b.WriteRune(r)
		}
		flush()
		parts[i] = b.String()
	}
	return strings.Join(parts, covered)
}

func isWordRune(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 0x7f
}

// maskValue masks a name, an address or an ID word by word, so an address
// keeps its @ and its dots and still reads as an address.
func maskValue(kind, value string) string {
	if kind == "email address" {
		local, domain, _ := strings.Cut(value, "@")
		return maskWord(local) + "@" + maskText(domain)
	}
	return maskText(value)
}
