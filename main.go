package main

// Accent says what would tie an anonymous project to its author's own.
//
// People who publish code under a name that is not theirs, the developer of
// a tool their government would rather did not exist, the researcher who
// does not want an exploit on their CV, tend to look after the obvious
// things: the git email, the account, the times of day. What they publish
// is still code, and code has an accent. The helper copied from the old
// project, the comment that explains the same workaround in the same words,
// the verbs and abbreviations somebody reaches for in every name they
// write. Each of these was something only a patient reader would notice.
// A language model notices them in seconds, and research in 2025 showed it
// attributing code to its author from a handful of examples.
//
// Given two folders, this says what ties them: the names and accounts both
// carry, the lines both have that nobody else does, and how much of the
// rare vocabulary of their names, comments and messages is the same. It
// only reads. Nothing is written, nothing is sent anywhere, and there is
// no model in it: the comparison is arithmetic anybody can redo by hand.

import (
	"flag"
	"fmt"
	"os"
)

var version = "unreleased"

const (
	exitApart  = 0
	exitTied   = 1 // the same names, copied lines, or one person's words
	exitBroken = 2
	exitUnsure = 3 // no code was read, which is not the same as no ties
)

func main() {
	reveal := flag.Bool("reveal", false, "print names, accounts and shared lines instead of masking them (secrets stay covered)")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println("Accent " + version)
		return
	}
	if flag.NArg() == 0 {
		usage()
		os.Exit(exitBroken)
	}

	var sources []*Source
	for _, where := range flag.Args() {
		sources = append(sources, readSource(where))
	}
	s := study(sources, *reveal)
	report := s.writeUp()
	report.render(os.Stdout)

	switch {
	case s.read() == 0:
		os.Exit(exitUnsure)
	case report.worst() >= Ties:
		os.Exit(exitTied)
	}
	os.Exit(exitApart)
}

func usage() {
	fmt.Fprintln(os.Stderr, `Accent - what ties your anonymous code to you

  accent named-project/ anonymous-project/   what ties the two together
  accent project/                            what in one project names you
  accent a/ b/ c/                            every pair compared
  accent --reveal ...                        print names and shared lines unmasked
  accent --version

It reads Go, Python, JavaScript, TypeScript, Rust, Java, C, C++, C#,
Kotlin, Swift, Ruby, PHP, shell and Lua, and leaves out what was not written
by hand: vendored libraries, generated code, minified bundles.

For two folders it reports the names, accounts and IDs both carry, the
lines, comments and messages both have that nobody else does, and the rare
words both use, set against what two strangers usually share.

Names and shared lines are masked unless you pass --reveal, so the report
can be pasted somewhere as it is. Secrets are covered either way.

It only reads. Nothing is written, nothing is sent anywhere, and there is
no model in it.`)
}
