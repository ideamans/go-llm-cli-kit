// Package skillcheck validates a distributable Claude Code plugin directory:
// the plugin manifest plus its Agent Skills.
//
// The rule it enforces above all others is that a *distributed* SKILL.md must
// stick to the open Agent Skills frontmatter (name, description, license,
// compatibility, metadata, allowed-tools). Claude Code extensions such as
// argument-hint or paths are silently ignored by Copilot, Cursor and Gemini CLI,
// so putting them in a skill we publish means rewriting it later. Claude-only
// behaviour belongs under metadata.claude-code.*, and project-local skills under
// .claude/skills are free to use whatever they like — those are not validated
// here.
//
// Typical use from a repository's own test suite:
//
//	func TestPluginSkills(t *testing.T) {
//	        report := skillcheck.CheckDir("plugins/gplay", skillcheck.Options{
//	                Version:  version.Version,
//	                Keywords: []string{"google play", "release"},
//	        })
//	        for _, p := range report.Problems {
//	                t.Error(p)
//	        }
//	}
package skillcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Standard Agent Skills frontmatter fields (agentskills.io/specification).
var standardFields = map[string]bool{
	"name":          true,
	"description":   true,
	"license":       true,
	"compatibility": true,
	"metadata":      true,
	"allowed-tools": true,
}

// Claude Code extensions. Harmless in .claude/skills, wrong in a skill we
// distribute to other agent hosts.
var claudeOnlyFields = map[string]bool{
	"disable-model-invocation": true,
	"user-invocable":           true,
	"argument-hint":            true,
	"arguments":                true,
	"paths":                    true,
	"model":                    true,
	"effort":                   true,
	"context":                  true,
	"agent":                    true,
	"hooks":                    true,
	"shell":                    true,
	"when_to_use":              true,
}

var nameRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Options tunes the checks.
type Options struct {
	// Version, when non-empty, must equal plugin.json's version field. Pass the
	// CLI's own version so a release cannot ship a stale plugin manifest.
	Version string

	// Keywords are discovery terms that must appear (case-insensitively) in at
	// least one skill description. They are what makes an agent reach for the
	// skill in the first place, so a typo here is a silent loss of traffic.
	Keywords []string

	// RequireInstallSkill demands a skill whose name ends in "-install".
	// Distributed CLI plugins need one: without it the first failure a user
	// hits is "command not found" with no way forward.
	RequireInstallSkill bool
}

// Skill is one parsed SKILL.md.
type Skill struct {
	Dir         string
	Path        string
	Name        string
	Description string
	Fields      map[string]string
}

// Plugin is a parsed .claude-plugin/plugin.json.
type Plugin struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Homepage    string   `json:"homepage"`
	Repository  string   `json:"repository"`
	License     string   `json:"license"`
	Keywords    []string `json:"keywords"`
}

// Report is the outcome of checking a plugin directory.
type Report struct {
	Plugin   *Plugin
	Skills   []Skill
	Problems []string
}

// OK reports whether the plugin directory passed every check.
func (r Report) OK() bool { return len(r.Problems) == 0 }

// Error returns the problems as a single error, or nil when there are none.
func (r Report) Error() error {
	if r.OK() {
		return nil
	}
	return fmt.Errorf("skillcheck: %s", strings.Join(r.Problems, "; "))
}

func (r *Report) failf(format string, args ...any) {
	r.Problems = append(r.Problems, fmt.Sprintf(format, args...))
}

// CheckDir validates a plugin directory such as plugins/gplay.
func CheckDir(dir string, opts Options) Report {
	var report Report

	plugin, err := readPlugin(filepath.Join(dir, ".claude-plugin", "plugin.json"))
	if err != nil {
		report.failf("%v", err)
	} else {
		report.Plugin = plugin
		checkPlugin(&report, dir, plugin, opts)
	}

	skillsDir := filepath.Join(dir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		report.failf("read %s: %v", skillsDir, err)
		return report
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		skill, err := ReadSkill(path)
		if err != nil {
			report.failf("%v", err)
			continue
		}
		checkSkill(&report, skill)
		report.Skills = append(report.Skills, skill)
	}

	if len(report.Skills) == 0 {
		report.failf("%s contains no skills", skillsDir)
		return report
	}

	checkDiscovery(&report, opts)
	return report
}

func readPlugin(path string) (*Plugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var p Plugin
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &p, nil
}

func checkPlugin(r *Report, dir string, p *Plugin, opts Options) {
	if p.Name == "" {
		r.failf("plugin.json: name is required")
	} else if base := filepath.Base(dir); p.Name != base {
		r.failf("plugin.json: name %q does not match directory %q", p.Name, base)
	}
	if p.Description == "" {
		r.failf("plugin.json: description is required")
	}
	if p.Version == "" {
		r.failf("plugin.json: version is required")
	}
	if opts.Version != "" {
		want := strings.TrimPrefix(opts.Version, "v")
		if got := strings.TrimPrefix(p.Version, "v"); got != want {
			r.failf("plugin.json: version %q does not match CLI version %q", p.Version, opts.Version)
		}
	}
}

// ReadSkill parses one SKILL.md.
func ReadSkill(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, fmt.Errorf("read %s: %w", path, err)
	}
	fields, err := parseFrontmatter(string(data))
	if err != nil {
		return Skill{}, fmt.Errorf("%s: %w", path, err)
	}
	return Skill{
		Dir:         filepath.Base(filepath.Dir(path)),
		Path:        path,
		Name:        fields["name"],
		Description: fields["description"],
		Fields:      fields,
	}, nil
}

func checkSkill(r *Report, s Skill) {
	switch {
	case s.Name == "":
		r.failf("%s: name is required", s.Path)
	case len(s.Name) > 64:
		r.failf("%s: name is %d characters (max 64)", s.Path, len(s.Name))
	case !nameRE.MatchString(s.Name):
		r.failf("%s: name %q must be lowercase kebab-case", s.Path, s.Name)
	case s.Name != s.Dir:
		r.failf("%s: name %q does not match directory %q", s.Path, s.Name, s.Dir)
	}

	switch n := len(s.Description); {
	case n == 0:
		r.failf("%s: description is required", s.Path)
	case n > 1024:
		r.failf("%s: description is %d characters (max 1024)", s.Path, n)
	case n < 40:
		r.failf("%s: description is only %d characters — say both what it does and when to use it", s.Path, n)
	}

	if c := s.Fields["compatibility"]; len(c) > 500 {
		r.failf("%s: compatibility is %d characters (max 500)", s.Path, len(c))
	}

	var offenders []string
	for field := range s.Fields {
		if standardFields[field] {
			continue
		}
		if claudeOnlyFields[field] {
			offenders = append(offenders, field)
			continue
		}
		offenders = append(offenders, field)
	}
	sort.Strings(offenders)
	for _, field := range offenders {
		if claudeOnlyFields[field] {
			r.failf("%s: %q is a Claude Code extension — move it under metadata.claude-code.*", s.Path, field)
			continue
		}
		r.failf("%s: %q is not an Agent Skills standard field", s.Path, field)
	}
}

func checkDiscovery(r *Report, opts Options) {
	if len(opts.Keywords) > 0 {
		var all strings.Builder
		for _, s := range r.Skills {
			all.WriteString(strings.ToLower(s.Description))
			all.WriteString("\n")
		}
		haystack := all.String()
		for _, kw := range opts.Keywords {
			if !strings.Contains(haystack, strings.ToLower(kw)) {
				r.failf("no skill description mentions the discovery keyword %q", kw)
			}
		}
	}

	if opts.RequireInstallSkill {
		found := false
		for _, s := range r.Skills {
			if strings.HasSuffix(s.Name, "-install") {
				found = true
				break
			}
		}
		if !found {
			r.failf("no *-install skill: users whose PATH lacks the CLI have no way forward")
		}
	}
}

// parseFrontmatter extracts the top-level keys of a YAML frontmatter block.
//
// Values are only ever compared, counted or reported, so this deliberately
// handles the flat subset the Agent Skills spec uses (scalars, quoted scalars,
// block scalars and nested blocks recorded as present) rather than pulling in a
// YAML dependency that every importing CLI would inherit.
func parseFrontmatter(content string) (map[string]string, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	lines := strings.Split(content, "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.TrimSpace(line) == "---" {
			start = i
		}
		break
	}
	if start < 0 {
		return nil, fmt.Errorf("missing YAML frontmatter")
	}

	end := -1
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, fmt.Errorf("unterminated YAML frontmatter")
	}

	fields := map[string]string{}
	var currentKey string
	var block []string

	flush := func() {
		if currentKey != "" {
			fields[currentKey] = strings.TrimSpace(strings.Join(block, " "))
		}
		currentKey, block = "", nil
	}

	for _, line := range lines[start+1 : end] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Continuation of the previous key: indented, or a list item.
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(strings.TrimSpace(line), "- ") {
			if currentKey != "" {
				block = append(block, strings.TrimSpace(line))
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		flush()
		currentKey = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if value == "|" || value == ">" || value == "|-" || value == ">-" {
			value = ""
		}
		block = []string{unquote(value)}
	}
	flush()

	return fields, nil
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
