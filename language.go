package main

// The languages Accent reads, and what it needs to know about each to take
// the code apart: where comments and strings start and end, which words are
// the language's own, and whether one formatter has decided the layout for
// everybody.

import (
	"path/filepath"
	"strings"
)

type Language struct {
	Name string
	Exts []string
	// Line comments start with one of these and run to the end of the
	// line. Block comments run between a pair.
	Line   []string
	Blocks [][2]string
	// Nested is set where block comments can contain block comments, which
	// is Rust and nothing else here. Reading /* /* */ as closed after the
	// first */ puts the rest of the comment into the code.
	Nested bool
	// Quotes open strings that a backslash escapes. Raw open strings that
	// end only at the same mark again, like Go's backtick or Python's three
	// quotes.
	Quotes string
	Raw    []string
	// Hash is true for languages where # starts a comment.
	Hash     bool
	Keywords map[string]bool
	// Formatter is the tool nearly everyone runs on this language, if
	// there is one. Where it is there, the layout of the code is the
	// formatter's and not the author's, and saying two people indent alike
	// would be saying they both ran gofmt.
	Formatter string
}

func wordSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(s) {
		out[w] = true
	}
	return out
}

var cLike = "break case const continue default do else for goto if return sizeof static struct switch typedef union void volatile while"

var languages = []*Language{
	{
		Name: "Go", Exts: []string{".go"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}},
		Quotes: `"'`, Raw: []string{"`"}, Formatter: "gofmt",
		Keywords: wordSet("break case chan const continue default defer else fallthrough for func go goto if import " +
			"interface map package range return select struct switch type var nil true false iota append len cap " +
			"make new panic recover close copy delete any"),
	},
	{
		Name: "Python", Exts: []string{".py", ".pyw"}, Quotes: `"'`, Raw: []string{`"""`, `'''`}, Hash: true,
		Keywords: wordSet("False None True and as assert async await break class continue def del elif else except " +
			"finally for from global if import in is lambda nonlocal not or pass raise return try while with yield " +
			"print len range isinstance super"),
	},
	{
		Name: "JavaScript", Exts: []string{".js", ".jsx", ".mjs", ".cjs"}, Line: []string{"//"},
		Blocks: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Raw: []string{"`"},
		Keywords: wordSet("break case catch class const continue debugger default delete do else export extends " +
			"finally for function if import in instanceof let new return super switch this throw try typeof var " +
			"void while with yield async await of null undefined true false require module exports"),
	},
	{
		Name: "TypeScript", Exts: []string{".ts", ".tsx", ".mts", ".cts"}, Line: []string{"//"},
		Blocks: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Raw: []string{"`"},
		Keywords: wordSet("break case catch class const continue debugger default delete do else export extends " +
			"finally for function if import in instanceof let new return super switch this throw try typeof var " +
			"void while with yield async await of null undefined true false interface type enum implements " +
			"private public protected readonly declare namespace abstract as any unknown never keyof infer is"),
	},
	{
		Name: "Rust", Exts: []string{".rs"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}}, Nested: true,
		Quotes: `"'`, Formatter: "rustfmt",
		Keywords: wordSet("as break const continue crate else enum extern false fn for if impl in let loop match mod " +
			"move mut pub ref return self Self static struct super trait true type unsafe use where while async " +
			"await dyn Some None Ok Err"),
	},
	{
		Name: "Java", Exts: []string{".java"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}},
		Quotes: `"'`, Raw: []string{`"""`},
		Keywords: wordSet("abstract assert boolean break byte case catch char class continue default do double else " +
			"enum extends final finally float for if implements import instanceof int interface long native new " +
			"package private protected public return short static super switch synchronized this throw throws " +
			"transient try void volatile while var record null true false"),
	},
	{
		Name: "C", Exts: []string{".c", ".h"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}},
		Quotes: `"'`,
		Keywords: wordSet(cLike + " auto char double enum extern float inline int long register restrict short " +
			"signed unsigned NULL"),
	},
	{
		Name: "C++", Exts: []string{".cc", ".cpp", ".cxx", ".hpp", ".hh", ".hxx"}, Line: []string{"//"},
		Blocks: [][2]string{{"/*", "*/"}}, Quotes: `"'`,
		Keywords: wordSet(cLike + " auto bool catch char class constexpr delete double enum explicit extern false " +
			"float friend inline int long mutable namespace new noexcept nullptr operator override private " +
			"protected public short signed static_cast template this throw true try typename unsigned using " +
			"virtual NULL"),
	},
	{
		Name: "C#", Exts: []string{".cs"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}},
		Quotes: `"'`,
		Keywords: wordSet("abstract as base bool break case catch class const continue default delegate do else enum " +
			"event false finally for foreach if in interface internal is new null namespace out override private " +
			"protected public readonly ref return sealed static string struct switch this throw true try typeof " +
			"using var virtual void while async await"),
	},
	{
		Name: "Kotlin", Exts: []string{".kt", ".kts"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}},
		Quotes: `"'`, Raw: []string{`"""`},
		Keywords: wordSet("as break class continue do else false for fun if in interface is null object package " +
			"return super this throw true try typealias val var when while private public internal override " +
			"data sealed companion lateinit"),
	},
	{
		Name: "Swift", Exts: []string{".swift"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}},
		Nested: true, Quotes: `"`, Raw: []string{`"""`},
		Keywords: wordSet("class deinit enum extension func import init let protocol struct subscript typealias var " +
			"break case continue default defer do else fallthrough for guard if in repeat return switch where " +
			"while as catch false is nil self Self super throw throws true try private public internal static"),
	},
	{
		Name: "Ruby", Exts: []string{".rb", ".rake"}, Quotes: `"'`, Hash: true,
		Keywords: wordSet("alias and begin break case class def defined? do else elsif end ensure false for if in " +
			"module next nil not or redo rescue retry return self super then true undef unless until when while " +
			"yield require attr_accessor puts"),
	},
	{
		Name: "PHP", Exts: []string{".php"}, Line: []string{"//"}, Blocks: [][2]string{{"/*", "*/"}},
		Quotes: `"'`, Hash: true,
		Keywords: wordSet("abstract array as break case catch class clone const continue declare default do echo " +
			"else elseif empty extends final finally for foreach function global if implements include instanceof " +
			"interface isset list namespace new null private protected public require return static switch " +
			"throw trait try unset use var while true false"),
	},
	{
		Name: "Shell", Exts: []string{".sh", ".bash", ".zsh"}, Quotes: `"'`, Hash: true,
		Keywords: wordSet("if then else elif fi case esac for while until do done in function select time return " +
			"exit local export readonly set echo printf"),
	},
	{
		Name: "Lua", Exts: []string{".lua"}, Line: []string{"--"}, Blocks: [][2]string{{"--[[", "]]"}},
		Quotes: `"'`,
		Keywords: wordSet("and break do else elseif end false for function goto if in local nil not or repeat " +
			"return then true until while"),
	},
}

var byExt = func() map[string]*Language {
	out := map[string]*Language{}
	for _, l := range languages {
		for _, e := range l.Exts {
			out[e] = l
		}
	}
	return out
}()

func languageOf(path string) *Language {
	return byExt[strings.ToLower(filepath.Ext(path))]
}
