package main

// The libraries a project uses.
//
// Everybody uses the popular ones, so sharing them says nothing. The rest is
// a taste: the small YAML library somebody found once and has used ever
// since, the testing helper nobody else has heard of. Two projects that
// reach for the same three unusual libraries were probably set up by the
// same hands, and a search for any one of them on a dependency graph lists
// both.

import (
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strings"
)

type Dep struct {
	Ecosystem string
	Name      string
}

var (
	goRequire   = regexp.MustCompile(`(?m)^\s*(?:require\s+)?([a-z0-9][\w.-]*\.[a-z]{2,}/[\w./-]+)\s+v\d`)
	pyRequire   = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9._-]*)\s*(?:\[[^\]]*\])?\s*(?:[<>=!~;]|$)`)
	tomlSection = regexp.MustCompile(`^\s*\[([^\]]+)\]\s*$`)
	tomlKey     = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)\s*=`)
	quoted      = regexp.MustCompile(`["']([A-Za-z0-9][A-Za-z0-9._-]*)\s*(?:\[[^\]]*\])?\s*(?:[<>=!~;][^"']*)?["']`)
	gem         = regexp.MustCompile(`^\s*gem\s+["']([A-Za-z0-9_.-]+)["']`)
	gradle      = regexp.MustCompile(`(?:implementation|api|compile|testImplementation|runtimeOnly)\s*\(?\s*["']([\w.-]+:[\w.-]+)`)
	pomArtifact = regexp.MustCompile(`<artifactId>([\w.-]+)</artifactId>`)
)

func depsOf(s *Source) []Dep {
	var out []Dep
	add := func(eco, name string) {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			out = append(out, Dep{eco, name})
		}
	}
	for _, f := range append(append([]*File(nil), s.Code...), s.Other...) {
		base := strings.ToLower(path.Base(f.Path))
		switch {
		case base == "go.mod":
			for _, m := range goRequire.FindAllStringSubmatch(f.Text, -1) {
				add("Go", m[1])
			}
		case base == "package.json":
			var manifest map[string]any
			if json.Unmarshal([]byte(f.Text), &manifest) != nil {
				continue
			}
			for _, field := range []string{"dependencies", "devDependencies", "peerDependencies", "optionalDependencies"} {
				if deps, ok := manifest[field].(map[string]any); ok {
					for name := range deps {
						add("npm", name)
					}
				}
			}
		case strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt"):
			for _, line := range strings.Split(f.Text, "\n") {
				line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
				if line == "" || strings.HasPrefix(line, "-") || strings.Contains(line, "://") {
					continue
				}
				if m := pyRequire.FindStringSubmatch(line); m != nil {
					add("PyPI", m[1])
				}
			}
		case base == "pyproject.toml", base == "cargo.toml":
			eco := "PyPI"
			if base == "cargo.toml" {
				eco = "crates"
			}
			section := ""
			inList := false
			for _, line := range strings.Split(f.Text, "\n") {
				if m := tomlSection.FindStringSubmatch(line); m != nil {
					section, inList = m[1], false
					continue
				}
				switch {
				case strings.HasSuffix(section, "dependencies"):
					if m := tomlKey.FindStringSubmatch(line); m != nil && m[1] != "python" {
						add(eco, m[1])
					}
				case section == "project":
					if strings.HasPrefix(strings.TrimSpace(line), "dependencies") {
						inList = true
					}
					if inList {
						for _, m := range quoted.FindAllStringSubmatch(line, -1) {
							add(eco, m[1])
						}
						if strings.Contains(line, "]") {
							inList = false
						}
					}
				}
			}
		case base == "gemfile":
			for _, line := range strings.Split(f.Text, "\n") {
				if m := gem.FindStringSubmatch(line); m != nil {
					add("RubyGems", m[1])
				}
			}
		case base == "composer.json":
			var manifest map[string]any
			if json.Unmarshal([]byte(f.Text), &manifest) != nil {
				continue
			}
			for _, field := range []string{"require", "require-dev"} {
				if deps, ok := manifest[field].(map[string]any); ok {
					for name := range deps {
						if name != "php" && !strings.HasPrefix(name, "ext-") {
							add("Packagist", name)
						}
					}
				}
			}
		case base == "pom.xml":
			for _, m := range pomArtifact.FindAllStringSubmatch(f.Text, -1) {
				add("Maven", m[1])
			}
		case base == "build.gradle", base == "build.gradle.kts":
			for _, m := range gradle.FindAllStringSubmatch(f.Text, -1) {
				add("Maven", m[1])
			}
		}
	}
	return distinctDeps(out)
}

func distinctDeps(ds []Dep) []Dep {
	seen := map[Dep]bool{}
	var out []Dep
	for _, d := range ds {
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ecosystem != out[j].Ecosystem {
			return out[i].Ecosystem < out[j].Ecosystem
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// sharedDeps is what both use that few projects do.
func sharedDeps(a, b []Dep) []Dep {
	in := map[Dep]bool{}
	for _, d := range a {
		in[d] = true
	}
	var out []Dep
	for _, d := range b {
		if in[d] && !commonDeps[hashOf(d.Ecosystem+"\x00"+d.Name)] {
			out = append(out, d)
		}
	}
	return out
}
