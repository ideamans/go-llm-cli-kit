package skillcheck_test

import (
	"testing"

	"github.com/ideamans/go-llm-cli-kit/skillcheck"
)

// The plugin template is copied into every CLI repository, so a template that
// fails the checks would propagate a broken plugin fifteen times over.
func TestShippedTemplatePasses(t *testing.T) {
	report := skillcheck.CheckDir("../templates/plugins/example-cli", skillcheck.Options{
		Version:             "0.1.0",
		Keywords:            []string{"install", "example-cli"},
		RequireInstallSkill: true,
	})
	for _, problem := range report.Problems {
		t.Error(problem)
	}
	if len(report.Skills) != 2 {
		t.Errorf("template ships %d skills, want usage + install", len(report.Skills))
	}
}
