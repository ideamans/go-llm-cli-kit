package llmcmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/spf13/cobra"

	"github.com/ideamans/go-llm-cli-kit/llmdocs"
)

func testConfig() Config {
	return Config{Docs: llmdocs.New(fstest.MapFS{
		"00-guide.md":    {Data: []byte("# Guide\n\nDrive the CLI like this.\n")},
		"90-commands.md": {Data: []byte("# Command catalog\n")},
	}, ".")}
}

func runRoot(t *testing.T, args ...string) string {
	t.Helper()
	root := &cobra.Command{Use: "demo", SilenceUsage: true, SilenceErrors: true}
	AddTo(root, testConfig())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return out.String()
}

func TestSubcommandPrintsMarkdown(t *testing.T) {
	out := runRoot(t, "llm")
	if !strings.Contains(out, "# Guide") || !strings.Contains(out, "# Command catalog") {
		t.Errorf("markdown output missing chapters:\n%s", out)
	}
}

func TestSubcommandJSONFormat(t *testing.T) {
	out := runRoot(t, "llm", "--format", "json")
	var sections []llmdocs.Section
	if err := json.Unmarshal([]byte(out), &sections); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(sections) != 2 || sections[0].Title != "Guide" {
		t.Errorf("unexpected sections: %+v", sections)
	}
}

func TestSubcommandRejectsUnknownFormat(t *testing.T) {
	root := &cobra.Command{Use: "demo", SilenceUsage: true, SilenceErrors: true}
	AddTo(root, testConfig())
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"llm", "--format", "yaml"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for an unknown format")
	}
	if !strings.Contains(err.Error(), "yaml") {
		t.Errorf("error should name the bad format: %v", err)
	}
}

func TestRenderFormats(t *testing.T) {
	cfg := testConfig()
	for _, format := range []string{"", "markdown", "md", "MD"} {
		out, err := Render(cfg, format)
		if err != nil {
			t.Fatalf("Render(%q): %v", format, err)
		}
		if !strings.HasPrefix(out, "# Guide") {
			t.Errorf("Render(%q) = %q", format, out)
		}
	}
	if _, err := Render(Config{}, "markdown"); err == nil {
		t.Error("expected an error when Docs is nil")
	}
}

func TestLegacyFlagAnywhereOnTheCommandLine(t *testing.T) {
	cases := [][]string{
		{"--llm"},
		{"apps", "list", "--llm"},
		{"--profile", "x", "--llm", "apps"},
	}
	for _, args := range cases {
		var out bytes.Buffer
		handled, err := HandleLegacy(args, testConfig(), &out)
		if err != nil {
			t.Fatalf("HandleLegacy(%v): %v", args, err)
		}
		if !handled {
			t.Fatalf("HandleLegacy(%v) did not handle the flag", args)
		}
		if !strings.Contains(out.String(), "# Guide") {
			t.Errorf("HandleLegacy(%v) printed %q", args, out.String())
		}
	}
}

func TestLegacyFlagIgnoredAfterDoubleDash(t *testing.T) {
	var out bytes.Buffer
	handled, err := HandleLegacy([]string{"run", "--", "--llm"}, testConfig(), &out)
	if err != nil {
		t.Fatalf("HandleLegacy: %v", err)
	}
	if handled {
		t.Error("--llm after -- is an operand, not a flag")
	}
}

func TestLegacyFlagAbsent(t *testing.T) {
	var out bytes.Buffer
	handled, err := HandleLegacy([]string{"apps", "list"}, testConfig(), &out)
	if err != nil {
		t.Fatalf("HandleLegacy: %v", err)
	}
	if handled || out.Len() != 0 {
		t.Errorf("handled=%v out=%q, want false and empty", handled, out.String())
	}
}

func TestAddToRegistersHiddenLegacyFlag(t *testing.T) {
	root := &cobra.Command{Use: "demo"}
	AddTo(root, testConfig())

	flag := root.PersistentFlags().Lookup("llm")
	if flag == nil {
		t.Fatal("--llm was not registered, so `demo --llm` would fail to parse")
	}
	if !flag.Hidden {
		t.Error("the deprecated --llm flag should be hidden from help")
	}
}

func TestAddToIsIdempotentWithAnExistingLegacyFlag(t *testing.T) {
	root := &cobra.Command{Use: "demo"}
	root.PersistentFlags().Bool("llm", false, "print detailed help for LLM agents")

	AddTo(root, testConfig()) // must not panic on redefinition

	if root.PersistentFlags().Lookup("llm") == nil {
		t.Error("pre-existing --llm flag disappeared")
	}
}
