package main

// What a finding is, and how it reaches the screen.

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type Severity int

const (
	Note Severity = iota
	// Hints narrow things down without settling them: libraries few
	// people use, a style too close to call.
	Hints
	// Ties is evidence somebody comparing the two would stop at: copied
	// lines, comments in the same words, one person's vocabulary.
	Ties
	// Names is the same account, address or ID in both. Nothing about it
	// is a matter of judgement.
	Names
)

type Finding struct {
	Tag    string
	Weight Severity
	Title  string
	Lines  []string
	// Advice is the part people act on. A finding without it is a finding
	// nobody does anything about.
	Advice string
}

type Report struct {
	Title    string
	Summary  []string
	Findings []Finding
	// Gaps are what could not be read, printed first, because a verdict
	// about half a folder is half a verdict.
	Gaps []string
}

func (r *Report) add(f Finding) { r.Findings = append(r.Findings, f) }

func (r *Report) worst() Severity {
	worst := Note
	for _, f := range r.Findings {
		worst = max(worst, f.Weight)
	}
	return worst
}

const (
	indent = "              "
	wrap   = 62
)

func (r *Report) render(out io.Writer) {
	fmt.Fprintf(out, "\n  %s\n  %s\n\n", r.Title, strings.Repeat("=", min(runes(r.Title), wrap+14)))
	for _, line := range r.Summary {
		fmt.Fprintf(out, "  %s\n", line)
	}
	if len(r.Summary) > 0 {
		fmt.Fprintln(out)
	}
	for _, g := range r.Gaps {
		block(out, "UNREAD ", "Something here could not be read", nil, g)
	}
	// Loudest first, keeping the order within a level, so somebody who
	// reads the first screen and stops has read the part that mattered.
	sorted := make([]Finding, len(r.Findings))
	copy(sorted, r.Findings)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Weight > sorted[j].Weight })
	for _, f := range sorted {
		block(out, f.Tag, f.Title, f.Lines, f.Advice)
	}
	fmt.Fprintln(out, "    Nothing was changed, and nothing was sent anywhere.")
	fmt.Fprintln(out)
}

func block(out io.Writer, tag, title string, lines []string, advice string) {
	head := wrapText(title, wrap+10)
	if len(head) == 0 {
		head = []string{""}
	}
	fmt.Fprintf(out, "    [%s] %s\n", tag, head[0])
	for _, line := range head[1:] {
		fmt.Fprintf(out, "%s%s\n", indent, line)
	}
	for _, line := range lines {
		if line = strings.TrimRight(line, " "); line == "" {
			fmt.Fprintln(out)
			continue
		}
		fmt.Fprintf(out, "%s%s\n", indent, line)
	}
	if advice != "" {
		if len(lines) > 0 {
			fmt.Fprintln(out)
		}
		for _, line := range wrapText(advice, wrap) {
			fmt.Fprintf(out, "%s%s\n", indent, line)
		}
	}
	fmt.Fprintln(out)
}

// wrapText counts letters rather than bytes, so a line with a mask or an
// accent in it wraps where it looks like it should.
func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if runes(line)+1+runes(w) > width {
			lines = append(lines, line)
			line = w
			continue
		}
		line += " " + w
	}
	return append(lines, line)
}

func runes(s string) int { return len([]rune(s)) }

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%s %s", commas(n), many)
}

// some is plural for a title, where "share a line" reads and "share 1
// line" reads like a form.
func some(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return commas(n) + " " + many
}

func commas(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// atMost keeps a list to a screen. It counts entries rather than lines:
// an entry that runs to two lines is still one thing found.
func atMost(entries [][]string, n int) []string {
	var lines []string
	for i, e := range entries {
		if i == n {
			lines = append(lines, fmt.Sprintf("... and %d more", len(entries)-n))
			break
		}
		lines = append(lines, e...)
	}
	return lines
}

// clip cuts a line down to something that fits, counting letters.
func clip(s string, width int) string {
	s = strings.Join(strings.Fields(s), " ")
	if runes(s) > width {
		s = string([]rune(s)[:width-3]) + "..."
	}
	return s
}
