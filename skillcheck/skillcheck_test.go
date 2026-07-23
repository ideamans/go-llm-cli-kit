package skillcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodDescription = "Render a chart to an image with the demo CLI. Use when the user asks to draw or plot data."

func writePlugin(t *testing.T, dir, manifest string, skills map[string]string) string {
	t.Helper()
	root := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(root, ".claude-plugin", "plugin.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range skills {
		skillDir := filepath.Join(root, "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const validManifest = `{
  "name": "demo",
  "description": "Demo plugin",
  "version": "1.2.0"
}`

func validSkill(name string) string {
	return "---\nname: " + name + "\ndescription: " + goodDescription +
		"\nlicense: MIT\ncompatibility: Requires the demo CLI on PATH.\n---\n\n# " + name + "\n"
}

func problemsContaining(r Report, substr string) bool {
	for _, p := range r.Problems {
		if strings.Contains(p, substr) {
			return true
		}
	}
	return false
}

func TestValidPluginPasses(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render":  validSkill("demo-render"),
		"demo-install": validSkill("demo-install"),
	})

	report := CheckDir(dir, Options{
		Version:             "1.2.0",
		Keywords:            []string{"chart", "plot"},
		RequireInstallSkill: true,
	})
	if !report.OK() {
		t.Fatalf("expected a clean report, got %v", report.Problems)
	}
	if report.Error() != nil {
		t.Errorf("Error() = %v, want nil", report.Error())
	}
	if len(report.Skills) != 2 {
		t.Errorf("got %d skills, want 2", len(report.Skills))
	}
}

func TestVersionMismatch(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render": validSkill("demo-render"),
	})

	report := CheckDir(dir, Options{Version: "1.3.0"})
	if !problemsContaining(report, "does not match CLI version") {
		t.Errorf("version drift not reported: %v", report.Problems)
	}

	// A leading v on either side is not a mismatch.
	if r := CheckDir(dir, Options{Version: "v1.2.0"}); !r.OK() {
		t.Errorf("v-prefixed version should match: %v", r.Problems)
	}
}

func TestClaudeOnlyFieldRejected(t *testing.T) {
	skill := "---\nname: demo-render\ndescription: " + goodDescription +
		"\nargument-hint: <file>\n---\n\n# demo-render\n"
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{"demo-render": skill})

	report := CheckDir(dir, Options{})
	if !problemsContaining(report, "argument-hint") {
		t.Fatalf("Claude-only field not reported: %v", report.Problems)
	}
	if !problemsContaining(report, "metadata.claude-code") {
		t.Errorf("problem should point at the fix: %v", report.Problems)
	}
}

func TestMetadataAndAllowedToolsAreStandard(t *testing.T) {
	skill := "---\nname: demo-render\ndescription: " + goodDescription +
		"\nallowed-tools: Bash(demo:*) Read Write\nmetadata:\n  claude-code:\n    argument-hint: <file>\n---\n\n# demo-render\n"
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{"demo-render": skill})

	if report := CheckDir(dir, Options{}); !report.OK() {
		t.Errorf("standard fields rejected: %v", report.Problems)
	}
}

func TestNameMustMatchDirectory(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render": validSkill("demo-renderer"),
	})
	if !problemsContaining(CheckDir(dir, Options{}), "does not match directory") {
		t.Error("name/directory mismatch not reported")
	}
}

func TestNameMustBeKebabCase(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"Demo_Render": validSkill("Demo_Render"),
	})
	if !problemsContaining(CheckDir(dir, Options{}), "kebab-case") {
		t.Error("non-kebab-case name not reported")
	}
}

func TestDescriptionLengthBounds(t *testing.T) {
	short := "---\nname: demo-render\ndescription: Renders.\n---\n\n# demo-render\n"
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{"demo-render": short})
	if !problemsContaining(CheckDir(dir, Options{}), "when to use it") {
		t.Error("too-short description not reported")
	}

	long := "---\nname: demo-render\ndescription: " + strings.Repeat("x", 1025) + "\n---\n\n# demo-render\n"
	dir = writePlugin(t, t.TempDir(), validManifest, map[string]string{"demo-render": long})
	if !problemsContaining(CheckDir(dir, Options{}), "max 1024") {
		t.Error("too-long description not reported")
	}
}

func TestMissingDescription(t *testing.T) {
	skill := "---\nname: demo-render\n---\n\n# demo-render\n"
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{"demo-render": skill})
	if !problemsContaining(CheckDir(dir, Options{}), "description is required") {
		t.Error("missing description not reported")
	}
}

func TestDiscoveryKeywords(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render": validSkill("demo-render"),
	})
	report := CheckDir(dir, Options{Keywords: []string{"chart", "sankey"}})
	if !problemsContaining(report, `"sankey"`) {
		t.Errorf("missing keyword not reported: %v", report.Problems)
	}
	if problemsContaining(report, `"chart"`) {
		t.Errorf("present keyword wrongly reported: %v", report.Problems)
	}
}

func TestRequireInstallSkill(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render": validSkill("demo-render"),
	})
	if !problemsContaining(CheckDir(dir, Options{RequireInstallSkill: true}), "-install skill") {
		t.Error("missing install skill not reported")
	}
}

func TestMissingFrontmatter(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render": "# demo-render\n\nNo frontmatter here.\n",
	})
	if !problemsContaining(CheckDir(dir, Options{}), "frontmatter") {
		t.Error("missing frontmatter not reported")
	}
}

func TestUnterminatedFrontmatter(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render": "---\nname: demo-render\n\n# demo-render\n",
	})
	if !problemsContaining(CheckDir(dir, Options{}), "unterminated") {
		t.Error("unterminated frontmatter not reported")
	}
}

func TestMissingManifestAndSkills(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), "", nil)
	report := CheckDir(dir, Options{})
	if !problemsContaining(report, "plugin.json") {
		t.Errorf("missing manifest not reported: %v", report.Problems)
	}
	if report.OK() {
		t.Error("expected failure")
	}
}

func TestPluginNameMustMatchDirectory(t *testing.T) {
	manifest := `{"name": "other", "description": "x", "version": "1.0.0"}`
	dir := writePlugin(t, t.TempDir(), manifest, map[string]string{
		"demo-render": validSkill("demo-render"),
	})
	if !problemsContaining(CheckDir(dir, Options{}), "does not match directory") {
		t.Error("plugin name/directory mismatch not reported")
	}
}

func TestQuotedAndBlockScalarValues(t *testing.T) {
	skill := "---\nname: \"demo-render\"\ndescription: >\n  " + goodDescription + "\n---\n\n# demo-render\n"
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{"demo-render": skill})
	if report := CheckDir(dir, Options{}); !report.OK() {
		t.Errorf("quoted/block scalars mis-parsed: %v", report.Problems)
	}
}

func TestReadSkill(t *testing.T) {
	dir := writePlugin(t, t.TempDir(), validManifest, map[string]string{
		"demo-render": validSkill("demo-render"),
	})
	skill, err := ReadSkill(filepath.Join(dir, "skills", "demo-render", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadSkill: %v", err)
	}
	if skill.Name != "demo-render" || skill.Dir != "demo-render" {
		t.Errorf("unexpected skill: %+v", skill)
	}
	if skill.Fields["license"] != "MIT" {
		t.Errorf("license = %q", skill.Fields["license"])
	}
}
