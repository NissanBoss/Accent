package main

// What a project says about whose it is.
//
// Not style: names. An email address in a test fixture, the account in a Go
// module path, the copyright line, the FUNDING.yml that lists a Ko-fi page,
// an analytics ID, a Docker Hub image under a username. Each one is a
// deterministic link: two projects that carry the same one belong to the
// same account, and no statistics come into it. These are the first thing
// anybody hunting for the person behind a repository looks for, which is
// why they are the first thing in the report.

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

type Leaf struct {
	Kind  string
	Value string
	Where Place
}

type leafRule struct {
	kind string
	// re has one group, around the part that names somebody.
	re *regexp.Regexp
	// hint is what a line has to contain, lower case, for the rule to be
	// worth running on it, with | between the choices.
	hint string
	// own restricts a rule to the files that say whose project this is:
	// the manifest, the README, the licence, the funding file. Run over
	// the code, a code host account is mostly the authors of the
	// libraries it imports, and two projects that both import cobra
	// would be tied together through spf13.
	own bool
}

var leafRules = []leafRule{
	{kind: "email address", re: regexp.MustCompile(`\b([A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,})\b`), hint: "@"},
	{kind: "code host account", re: regexp.MustCompile(`(?i)\b(?:github\.com|gitlab\.com|bitbucket\.org|codeberg\.org|sr\.ht)[/:]~?([A-Za-z0-9][A-Za-z0-9_.-]*)`), hint: "github|gitlab|bitbucket|codeberg|sr.ht", own: true},
	{kind: "GitHub Pages site", re: regexp.MustCompile(`(?i)\b([a-z0-9][a-z0-9-]*)\.github\.io\b`), hint: ".github.io"},
	{kind: "social account", re: regexp.MustCompile(`(?i)\b(?:twitter\.com|x\.com|instagram\.com|t\.me|youtube\.com/(?:c|channel|user)|medium\.com|dev\.to|keybase\.io|linkedin\.com/in|reddit\.com/u(?:ser)?)/@?([A-Za-z0-9_.-]{2,})`), hint: "twitter|x.com|instagram|t.me|youtube|medium|dev.to|keybase|linkedin|reddit"},
	{kind: "funding account", re: regexp.MustCompile(`(?i)\b(?:ko-fi\.com|patreon\.com|buymeacoffee\.com|liberapay\.com|opencollective\.com|github\.com/sponsors|paypal\.me)/([A-Za-z0-9_.-]{2,})`), hint: "ko-fi|patreon|buymeacoffee|liberapay|opencollective|sponsors|paypal.me"},
	{kind: "funding account", re: regexp.MustCompile(`^\s*(?:github|patreon|ko_fi|liberapay|open_collective|buy_me_a_coffee|custom)\s*:\s*\[?\s*["']?([A-Za-z0-9_.@/:-]{2,})`), hint: "github:|patreon:|ko_fi:|liberapay:|open_collective:|buy_me_a_coffee:|custom:", own: true},
	{kind: "Docker image account", re: regexp.MustCompile(`(?i)(?:docker\.io/|ghcr\.io/|quay\.io/|image:\s*["']?)([a-z0-9][a-z0-9_.-]{1,30})/[a-z0-9][a-z0-9_.-]*`), hint: "docker.io/|ghcr.io/|quay.io/|image:"},
	{kind: "npm scope", re: regexp.MustCompile(`"name"\s*:\s*"@([a-z0-9][a-z0-9_.-]*)/`), hint: `"name"`, own: true},
	{kind: "home folder", re: regexp.MustCompile(`(?:/home|/Users)/([A-Za-z0-9._-]+)`), hint: "/home/|/users/"},
	{kind: "home folder", re: regexp.MustCompile(`(?i)\b[A-Z]:\\Users\\([^\\\s"']+)`), hint: `:\users\`},
	// The year or the (c) is required: the MIT licence says "COPYRIGHT
	// HOLDERS BE LIABLE" in capitals further down, and that is not a name.
	{kind: "copyright holder", re: regexp.MustCompile(`(?i)copyright\s+(?:(?:\(c\)|©)\s*(?:\d{4}(?:\s*[-,]\s*\d{4})*\s+)?|\d{4}(?:\s*[-,]\s*\d{4})*\s+)([A-Z][^\n<>()]{2,60}?)\s*(?:<|\(|\.?\s*$|\.?\s*all rights)`), hint: "copyright"},
	{kind: "author", re: regexp.MustCompile(`(?i)(?:@author|__author__\s*=|\bauthors?\s*[:=]\s*\[?|"author"\s*:\s*\{?\s*(?:"name"\s*:\s*)?)\s*["']?([A-Z][^"'\n<>,\]]{2,60})`), hint: "author"},
	{kind: "analytics ID", re: regexp.MustCompile(`\b((?:G|UA|GTM|AW)-[A-Z0-9]{6,}(?:-\d+)?)\b`), hint: "g-|ua-|gtm-|aw-"},
	{kind: "ad account", re: regexp.MustCompile(`\b((?:ca-)?pub-\d{10,20})\b`), hint: "pub-"},
	{kind: "Sentry account", re: regexp.MustCompile(`https://[0-9a-f]+@(o\d+)\.ingest\.(?:[a-z]+\.)?sentry\.io/\d+`), hint: "sentry.io"},
	{kind: "Firebase project", re: regexp.MustCompile(`(?i)\b([a-z0-9-]+)\.(?:firebaseapp\.com|firebaseio\.com|web\.app)\b`), hint: "firebase|web.app"},
	{kind: "Supabase project", re: regexp.MustCompile(`\b([a-z0-9]{20})\.supabase\.co\b`), hint: "supabase.co"},
	{kind: "hosted app", re: regexp.MustCompile(`(?i)\b([a-z0-9][a-z0-9-]*)\.(?:herokuapp\.com|netlify\.app|vercel\.app|fly\.dev|pages\.dev|onrender\.com|glitch\.me|replit\.app)\b`), hint: "herokuapp|netlify|vercel|fly.dev|pages.dev|onrender|glitch|replit"},
	{kind: "Cloudinary account", re: regexp.MustCompile(`res\.cloudinary\.com/([a-z0-9_-]+)/`), hint: "cloudinary"},
	{kind: "Stripe account", re: regexp.MustCompile(`\b(pk_(?:live|test)_[A-Za-z0-9]{20,})\b`), hint: "pk_"},
	{kind: "Mapbox account", re: regexp.MustCompile(`\b(pk\.eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,})`), hint: "pk.eyj"},
	{kind: "AWS account", re: regexp.MustCompile(`\barn:aws[a-z-]*:[a-z0-9-]*:[a-z0-9-]*:(\d{12}):`), hint: "arn:aws"},
	{kind: "S3 bucket", re: regexp.MustCompile(`s3://([a-z0-9][a-z0-9.-]{2,62})`), hint: "s3://"},
	{kind: "S3 bucket", re: regexp.MustCompile(`\b([a-z0-9][a-z0-9.-]{2,62})\.s3[.-](?:[a-z0-9-]+\.)?amazonaws\.com`), hint: "amazonaws.com"},
	{kind: "Bitcoin address", re: regexp.MustCompile(`\b(bc1[ac-hj-np-z02-9]{25,62}|[13][a-km-zA-HJ-NP-Z1-9]{25,34})\b`), hint: "bc1|donat|btc|bitcoin|wallet"},
	{kind: "Ethereum address", re: regexp.MustCompile(`\b(0x[a-fA-F0-9]{40})\b`), hint: "donat|wallet|ethereum|tips"},
	{kind: "PGP key", re: regexp.MustCompile(`\b([A-F0-9]{4}(?:\s?[A-F0-9]{4}){9})\b`), hint: "pgp|gpg|fingerprint"},
	{kind: "Matrix account", re: regexp.MustCompile(`(@[a-z0-9._=-]+:[a-z0-9.-]+\.[a-z]{2,})\b`), hint: "matrix|element|@"},
	{kind: "Discord server", re: regexp.MustCompile(`(?i)\bdiscord(?:\.gg|app\.com/invite|\.com/invite)/([A-Za-z0-9-]{4,})`), hint: "discord"},
}

// Values that are nobody in particular: placeholders, the examples in the
// documentation of every tool, the pages every project links to.
var nobody = wordSet("example examples user username yourname your-name you me name foo bar baz test " +
	"tests github gitlab bitbucket docs issues pulls orgs sponsors features about login settings " +
	"marketplace topics explore trending security apps actions blog www api raw gist gists " +
	"golang google microsoft facebook twitter apple amazon aws torvalds kubernetes docker library " +
	"rust-lang python nodejs npm denoland vuejs vercel mozilla apache gnu kde gnome freebsd " +
	"openssl curl git moby hashicorp jetbrains")

// ownFile says whether a file is one of those that say whose a project is.
func ownFile(rel string) bool {
	base := strings.ToLower(path.Base(rel))
	switch {
	case strings.HasPrefix(base, "readme"), strings.HasPrefix(base, "license"),
		strings.HasPrefix(base, "licence"), strings.HasPrefix(base, "copying"),
		strings.HasPrefix(base, "contributing"), strings.HasPrefix(base, "authors"),
		strings.HasPrefix(base, "maintainers"), base == "funding.yml", base == "go.mod",
		base == "package.json", base == "cargo.toml", base == "pyproject.toml", base == "setup.py",
		base == "setup.cfg", base == "composer.json", base == "pom.xml", base == "build.gradle",
		strings.HasSuffix(base, ".gemspec"), strings.HasSuffix(base, ".cabal"),
		strings.HasPrefix(rel, ".github/"):
		return true
	}
	return false
}

func leavesOf(s *Source) []Leaf {
	var out []Leaf
	for _, f := range append(append([]*File(nil), s.Code...), s.Other...) {
		own := ownFile(f.Path)
		for i, line := range strings.Split(f.Text, "\n") {
			lower := strings.ToLower(line)
			for _, r := range leafRules {
				if r.own && !own || !mentions(lower, r.hint) {
					continue
				}
				if r.kind == "code host account" && !ownerLine(f.Path, line) {
					continue
				}
				for _, at := range r.re.FindAllStringSubmatchIndex(line, -1) {
					// In scheme://user:password@host the password and the
					// host look like an address. They are the one thing on
					// the line that must never be printed.
					if r.kind == "email address" && at[2] > 0 && line[at[2]-1] == ':' {
						continue
					}
					value := strings.TrimSpace(strings.TrimRight(line[at[2]:at[3]], ".,;:"))
					if worthNaming(r.kind, value) {
						out = append(out, Leaf{Kind: r.kind, Value: value, Where: Place{f.Path, i + 1}})
					}
				}
			}
		}
	}
	return out
}

func worthNaming(kind, value string) bool {
	lower := strings.ToLower(value)
	if len(value) < 2 || nobody[lower] || commonLeaves[hashOf(kind+"\x00"+lower)] {
		return false
	}
	switch kind {
	case "email address":
		at := strings.LastIndexByte(lower, '@')
		domain, local := lower[at+1:], lower[:at]
		if strings.HasSuffix(domain, "example.com") || strings.HasSuffix(domain, "example.org") ||
			strings.HasSuffix(domain, ".test") || strings.HasSuffix(domain, ".example") ||
			strings.HasSuffix(domain, ".local") || local == "noreply" || local == "no-reply" ||
			local == "git" || nobody[local] {
			return false
		}
		// A package at a version or an image at a digest looks like an
		// address and ends in a number.
		if strings.ContainsAny(domain[strings.LastIndexByte(domain, '.')+1:], "0123456789") {
			return false
		}
	case "copyright holder", "author":
		if strings.Contains(lower, "contributors") || strings.Contains(lower, "authors") ||
			strings.HasPrefix(lower, "the ") || strings.Contains(lower, "notice") {
			return false
		}
	}
	return true
}

// distinctLeaves keeps one of each value, in a stable order, with the first
// place it was seen.
func distinctLeaves(ls []Leaf) []Leaf {
	seen := map[string]bool{}
	var out []Leaf
	for _, l := range ls {
		key := l.Kind + "\x00" + strings.ToLower(l.Value)
		if !seen[key] {
			seen[key] = true
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return strings.ToLower(out[i].Value) < strings.ToLower(out[j].Value)
	})
	return out
}

// sharedLeavesOf pairs up what two projects both carry. What is everybody's
// is checked again here and not only when reading, because the tables can
// change between the two, which the calibration relies on.
func sharedLeavesOf(a, b []Leaf) [][2]Leaf {
	in := map[string]Leaf{}
	for _, l := range b {
		in[l.Kind+"\x00"+strings.ToLower(l.Value)] = l
	}
	var out [][2]Leaf
	for _, l := range a {
		key := l.Kind + "\x00" + strings.ToLower(l.Value)
		if other, ok := in[key]; ok && !commonLeaves[hashOf(key)] {
			out = append(out, [2]Leaf{l, other})
		}
	}
	return out
}

// ownerLine says whether a line in one of the files that say whose a
// project is names the project's own account rather than somebody else's.
// go.mod names its module once and then every module it requires; a README
// links to the projects it builds on and compares itself with. Only the
// lines that point at the project itself count: the module line, the
// repository and homepage fields of a manifest, the funding file, the
// workflows.
func ownerLine(rel, line string) bool {
	base := strings.ToLower(path.Base(rel))
	lower := strings.ToLower(line)
	switch {
	case base == "go.mod":
		return strings.HasPrefix(line, "module ")
	case base == "funding.yml", strings.HasPrefix(rel, ".github/"):
		return true
	case strings.HasPrefix(base, "readme"), strings.HasPrefix(base, "contributing"):
		// The clone line and the badges point at the project itself.
		return strings.Contains(lower, "git clone") || strings.Contains(lower, "badge") ||
			strings.Contains(lower, "go install") || strings.Contains(lower, "go get")
	}
	for _, field := range []string{"repository", "homepage", "url", "bugs", "source", "documentation", "author"} {
		if strings.Contains(lower, field) {
			return true
		}
	}
	return false
}
