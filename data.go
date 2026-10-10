package main

// The tables built from the calibration corpus, read once when the program
// starts.
//
// Each says what is everybody's rather than anybody's: the lines, names and
// libraries that turned up in the projects of three or more different
// people, how often each word is used, what two strangers' projects
// usually share, and how the people in the corpus write on average,
// language by language. The lines, names and words are kept as hashes.
// They came out of other people's code, and a table of them in plain text
// would include other people's email addresses.

import (
	"bufio"
	"compress/gzip"
	"embed"
	"strconv"
	"strings"
)

//go:embed data/*.tsv.gz
var tables embed.FS

var (
	// commonTexts holds lines, comments and messages; commonLeaves holds
	// names, accounts and IDs; commonDeps holds libraries. The Docker
	// images of bitnami and the GitHub account of a popular library's
	// author are in the corpus three times over, and nobody's in
	// particular.
	commonTexts  = map[uint64]bool{}
	commonLeaves = map[uint64]bool{}
	commonDeps   = map[uint64]bool{}
	// usual is, for each language and feature, how often the people in the
	// corpus use it: the mean of their rates and how much they differ.
	usual = map[string]map[string][2]float64{}
	// wordTables is, for each language, how many projects use each word.
	wordTables = map[string]*wordTable{}
	// strangers is, for each language, what pairs of projects by
	// different people in the corpus share, sorted: two hundred points
	// from the least to the most.
	strangers = map[string][]float64{}
)

// wordTable is, for one language, how many projects in the corpus use each
// word, by hash, and how many projects there were.
type wordTable struct {
	people int
	df     map[uint64]int
}

func init() {
	readHashes("data/common-texts.tsv.gz", commonTexts)
	readHashes("data/common-leaves.tsv.gz", commonLeaves)
	readTable("data/common-deps.tsv.gz", func(f []string) {
		if len(f) == 2 {
			commonDeps[hashOf(f[0]+"\x00"+f[1])] = true
		}
	})
	readTable("data/usual.tsv.gz", func(f []string) {
		if len(f) != 4 {
			return
		}
		mean, _ := strconv.ParseFloat(f[2], 64)
		sd, _ := strconv.ParseFloat(f[3], 64)
		if usual[f[0]] == nil {
			usual[f[0]] = map[string][2]float64{}
		}
		usual[f[0]][f[1]] = [2]float64{mean, sd}
	})
	readTable("data/words.tsv.gz", func(f []string) {
		if len(f) != 3 {
			return
		}
		if f[0] == "people" {
			n, _ := strconv.Atoi(f[2])
			table(f[1]).people = n
			return
		}
		h, err := strconv.ParseUint(f[1], 16, 64)
		n, _ := strconv.Atoi(f[2])
		if err == nil {
			table(f[0]).df[h] = n
		}
	})
	if len(wordTables) > 0 {
		wordTables["*"] = pooledTable(wordTables)
	}
	readTable("data/strangers.tsv.gz", func(f []string) {
		if len(f) != 2 {
			return
		}
		for _, v := range strings.Split(f[1], ",") {
			if x, err := strconv.ParseFloat(v, 64); err == nil {
				strangers[f[0]] = append(strangers[f[0]], x)
			}
		}
	})
}

func table(lang string) *wordTable {
	if wordTables[lang] == nil {
		wordTables[lang] = &wordTable{df: map[uint64]int{}}
	}
	return wordTables[lang]
}

func readHashes(name string, into map[uint64]bool) {
	readTable(name, func(f []string) {
		if h, err := strconv.ParseUint(f[0], 16, 64); err == nil {
			into[h] = true
		}
	})
}

// readTable reads a gzipped table of tab separated rows. A table that is
// missing or broken leaves its map empty rather than stopping the program:
// the report is then noisier, not wrong.
func readTable(name string, row func([]string)) {
	f, err := tables.Open(name)
	if err != nil {
		return
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return
	}
	lines := bufio.NewScanner(z)
	lines.Buffer(make([]byte, 1<<16), 1<<20)
	for lines.Scan() {
		if line := lines.Text(); line != "" && !strings.HasPrefix(line, "#") {
			row(strings.Split(line, "\t"))
		}
	}
}
