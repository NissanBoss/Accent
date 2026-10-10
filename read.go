package main

// Finding the code that is somebody's own in a folder.
//
// Most of what sits in a project folder was not written by the person whose
// project it is: vendored libraries, node_modules, generated protobuf code,
// a minified bundle, a lock file. Measuring the style of any of it measures
// somebody else, and two projects that vendor the same library would look
// like the same author. So the reading leaves out what it can recognise as
// not written by hand, and the report says how much that was.

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type File struct {
	Path string
	Lang *Language
	Text string
}

type Source struct {
	Name string
	Root string
	// Code is what the style is measured on. Other is everything else
	// worth reading for shared lines and for names: configuration, build
	// scripts, manifests, documentation.
	Code  []*File
	Other []*File
	// Left is what was not read, by reason, so the report can say how much
	// of the folder the answer is about.
	Left map[string]int
	Gaps []string
}

// Folders that hold other people's code, or code nobody typed.
var notTheirs = map[string]bool{
	"node_modules": true, "vendor": true, "third_party": true, "third-party": true,
	"thirdparty": true, "external": true, "extern": true, "deps": true, "bower_components": true,
	"dist": true, "build": true, "out": true, "target": true, "obj": true, "__pycache__": true,
	"venv": true, "env": true, "site-packages": true, "Pods": true, "Carthage": true,
	"DerivedData": true, "coverage": true, "htmlcov": true, "generated": true, "gen": true,
	"migrations": true,
}

// Libraries people copy into a project by hand rather than install, by the
// start of their file names. A static/ folder is often the author's own
// JavaScript next to a copy of jQuery, so the folder is read and the copy
// is not.
var copiedLibraries = []string{
	"jquery", "bootstrap", "popper", "lodash", "underscore", "moment", "d3.", "three.", "chart.",
	"vue.", "react.", "react-dom", "angular.", "backbone", "handlebars", "highlight.", "prism.",
	"socket.io", "axios.", "sweetalert", "select2", "datatables", "fontawesome", "materialize",
}

// Files worth reading for shared lines and names even though their style
// is not measured.
var otherText = map[string]bool{
	".yml": true, ".yaml": true, ".toml": true, ".json": true, ".ini": true, ".cfg": true,
	".conf": true, ".md": true, ".txt": true, ".rst": true, ".xml": true, ".gradle": true,
	".properties": true, ".env": true, ".mk": true, ".cmake": true, ".html": true, ".css": true,
	".scss": true, ".sql": true,
}

var otherNames = map[string]bool{
	"Makefile": true, "Dockerfile": true, "Gemfile": true, "Rakefile": true, "Procfile": true,
	"Justfile": true, "Vagrantfile": true, "CMakeLists.txt": true, ".gitignore": true,
	".editorconfig": true, ".dockerignore": true, "go.mod": true, "Cargo.toml": true,
	"package.json": true, "pyproject.toml": true, "setup.py": true, "setup.cfg": true,
	"requirements.txt": true, "LICENSE": true, "LICENSE.md": true, "COPYING": true,
}

const tooBig = 1 << 20

func readSource(where string) *Source {
	s := &Source{Name: filepath.Base(strings.TrimRight(where, `/\`)), Left: map[string]int{}}
	stat, err := os.Stat(where)
	if err != nil {
		s.Gaps = append(s.Gaps, err.Error())
		return s
	}
	if !stat.IsDir() {
		s.readFile(where, filepath.Base(where))
		return s
	}
	s.Root = where
	var files []string
	err = filepath.WalkDir(where, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			s.Gaps = append(s.Gaps, err.Error())
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != where && (notTheirs[name] || strings.HasPrefix(name, ".") && name != ".github") {
				if !strings.HasPrefix(name, ".") {
					s.Left["vendored"]++
				}
				return fs.SkipDir
			}
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		s.Gaps = append(s.Gaps, err.Error())
	}
	sort.Strings(files)
	for _, f := range files {
		rel, _ := filepath.Rel(where, f)
		s.readFile(f, filepath.ToSlash(rel))
	}
	return s
}

func (s *Source) readFile(path, rel string) {
	lang := languageOf(path)
	base := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))
	if lang == nil && !otherText[ext] && !otherNames[base] {
		return
	}
	stat, err := os.Stat(path)
	if err != nil {
		s.Gaps = append(s.Gaps, err.Error())
		return
	}
	if stat.Size() > tooBig {
		s.Left["big"]++
		return
	}
	body, err := os.ReadFile(path)
	if err != nil {
		s.Gaps = append(s.Gaps, fmt.Sprintf("%s: %v", rel, err))
		return
	}
	if bytes.IndexByte(body, 0) >= 0 {
		s.Left["binary"]++
		return
	}
	text := string(body)
	if lang == nil {
		if !lockFile(base) {
			s.Other = append(s.Other, &File{Path: rel, Text: text})
		}
		return
	}
	if why := notWritten(rel, text); why != "" {
		s.Left[why]++
		return
	}
	s.Code = append(s.Code, &File{Path: rel, Lang: lang, Text: text})
}

func lockFile(name string) bool {
	switch name {
	case "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "Cargo.lock", "go.sum", "poetry.lock",
		"Pipfile.lock", "composer.lock", "Gemfile.lock", "npm-shrinkwrap.json":
		return true
	}
	return false
}

// notWritten says why a code file was not typed by a person, if it was not:
// a generated header, a minified bundle, a name that only generators use.
func notWritten(rel, text string) string {
	lower := strings.ToLower(rel)
	base := filepath.Base(lower)
	for _, lib := range copiedLibraries {
		if strings.HasPrefix(base, lib) {
			return "library"
		}
	}
	for _, suffix := range []string{".min.js", ".pb.go", "_pb2.py", "_pb2_grpc.py", ".pb.cc", ".pb.h",
		".generated.cs", ".g.dart", "_generated.go", ".bundle.js", "-bundle.js"} {
		if strings.HasSuffix(lower, suffix) {
			return "generated"
		}
	}
	head := text[:min(len(text), 1024)]
	if strings.Contains(head, "Code generated") && strings.Contains(head, "DO NOT EDIT") ||
		strings.Contains(head, "@generated") || strings.Contains(strings.ToLower(head), "auto-generated") ||
		strings.Contains(strings.ToLower(head), "autogenerated") {
		return "generated"
	}
	lines := strings.Count(text, "\n") + 1
	if len(text) > 5000 && len(text)/lines > 300 {
		return "minified"
	}
	return ""
}
