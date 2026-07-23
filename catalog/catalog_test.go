package catalog

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func demoRoot() *cobra.Command {
	root := &cobra.Command{Use: "demo", Short: "demo CLI"}
	root.PersistentFlags().String("profile", "", "profile name")

	apps := &cobra.Command{Use: "apps", Short: "manage apps"}
	list := &cobra.Command{
		Use:     "list [query]",
		Short:   "list apps",
		Long:    "List every app visible to the credentials.",
		Example: "demo apps list --limit 10",
		Aliases: []string{"ls"},
	}
	list.Flags().IntP("limit", "n", 50, "maximum rows")
	apps.AddCommand(list)

	hidden := &cobra.Command{Use: "secret", Short: "hidden", Hidden: true}
	deprecated := &cobra.Command{Use: "old", Short: "old", Deprecated: "use apps"}
	llm := &cobra.Command{Use: "llm", Short: "print the reference"}

	root.AddCommand(apps, hidden, deprecated, llm)
	return root
}

func TestBuildTree(t *testing.T) {
	c := Build(demoRoot(), Options{Skip: []string{"llm"}})

	if c.Path != "demo" {
		t.Errorf("root path = %q", c.Path)
	}
	if len(c.Commands) != 1 {
		var names []string
		for _, sub := range c.Commands {
			names = append(names, sub.Path)
		}
		t.Fatalf("got subcommands %v, want only demo apps", names)
	}
	apps := c.Commands[0]
	if apps.Path != "demo apps" {
		t.Fatalf("subcommand path = %q", apps.Path)
	}
	if len(apps.Commands) != 1 || apps.Commands[0].Path != "demo apps list" {
		t.Fatalf("unexpected leaf: %+v", apps.Commands)
	}
}

func TestPersistentFlagsReportedOnceAtRoot(t *testing.T) {
	c := Build(demoRoot(), Options{Skip: []string{"llm"}})

	if len(c.PersistentFlags) != 1 || c.PersistentFlags[0].Name != "profile" {
		t.Fatalf("root persistent flags = %+v", c.PersistentFlags)
	}
	leaf := c.Commands[0].Commands[0]
	for _, f := range leaf.Flags {
		if f.Name == "profile" {
			t.Error("inherited flag repeated on the leaf command")
		}
	}
	if len(leaf.Flags) != 1 || leaf.Flags[0].Name != "limit" {
		t.Fatalf("leaf flags = %+v", leaf.Flags)
	}
	if got := leaf.Flags[0]; got.Shorthand != "n" || got.Type != "int" || got.Default != "50" {
		t.Errorf("flag metadata = %+v", got)
	}
}

func TestHiddenAndDeprecatedExcludedByDefault(t *testing.T) {
	md := Markdown(demoRoot(), Options{Skip: []string{"llm"}})
	if strings.Contains(md, "secret") {
		t.Error("hidden command leaked into the catalog")
	}
	if strings.Contains(md, "demo old") {
		t.Error("deprecated command leaked into the catalog")
	}

	withHidden := Markdown(demoRoot(), Options{Skip: []string{"llm"}, IncludeHidden: true})
	if !strings.Contains(withHidden, "demo secret") {
		t.Error("IncludeHidden did not include the hidden command")
	}
}

func TestHelpAndCompletionAlwaysSkipped(t *testing.T) {
	root := demoRoot()
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	md := Markdown(root, Options{})
	if strings.Contains(md, "demo help") || strings.Contains(md, "demo completion") {
		t.Errorf("cobra built-ins leaked into the catalog:\n%s", md)
	}
}

func TestMarkdownShape(t *testing.T) {
	md := Markdown(demoRoot(), Options{
		Title: "Command catalog",
		Intro: "Every command demo accepts.",
		Skip:  []string{"llm"},
	})

	for _, want := range []string{
		"# Command catalog",
		"Every command demo accepts.",
		"## Global flags",
		"`--profile`",
		"## `demo apps`",
		"### `demo apps list`",
		"List every app visible to the credentials.",
		"demo apps list [query]", // usage line rewritten to the full path
		"Aliases: ls",
		"demo apps list --limit 10",
		"`-n`, `--limit`",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("catalog missing %q:\n%s", want, md)
		}
	}
}

func TestMarkdownEscapesPipesInFlagUsage(t *testing.T) {
	root := &cobra.Command{Use: "demo"}
	root.Flags().String("format", "markdown", "output format: markdown | json")
	sub := &cobra.Command{Use: "run", Short: "run"}
	sub.Flags().String("mode", "", "a | b")
	root.AddCommand(sub)

	md := Markdown(root, Options{})
	if strings.Contains(md, "a | b") {
		t.Errorf("pipe in flag usage was not escaped:\n%s", md)
	}
	if !strings.Contains(md, `a \| b`) {
		t.Errorf("expected escaped pipe:\n%s", md)
	}
}

func TestJSONIsStable(t *testing.T) {
	first, err := JSON(demoRoot(), Options{Skip: []string{"llm"}})
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	second, err := JSON(demoRoot(), Options{Skip: []string{"llm"}})
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if string(first) != string(second) {
		t.Error("catalog JSON is not deterministic across runs")
	}

	var c Command
	if err := json.Unmarshal(first, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Path != "demo" {
		t.Errorf("path = %q", c.Path)
	}
}

func TestMarkdownIsDeterministic(t *testing.T) {
	a := Markdown(demoRoot(), Options{})
	b := Markdown(demoRoot(), Options{})
	if a != b {
		t.Error("Markdown output differs between runs — go generate would churn the diff")
	}
}

func TestOmitFlags(t *testing.T) {
	md := Markdown(demoRoot(), Options{OmitFlags: true})
	if strings.Contains(md, "| flag |") {
		t.Errorf("OmitFlags still rendered a flag table:\n%s", md)
	}
}
