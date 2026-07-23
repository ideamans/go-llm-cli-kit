// Package catalog turns a live cobra command tree into a Markdown or JSON
// command catalog for LLM consumption.
//
// The point is that the catalog never drifts from the CLI: it is derived from
// the same *cobra.Command values the binary dispatches on. Projects run it from
// go generate and commit the result into their llmdocs directory, so the
// embedded reference is regenerated whenever commands or flags change.
//
//	//go:generate go run ./internal/gen-llmdocs
//
// CI then re-runs go generate and fails on `git diff --exit-code`.
package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Options controls catalog rendering.
type Options struct {
	// Title is the level-1 heading of the Markdown output.
	// Defaults to "# Command catalog".
	Title string

	// Intro is optional prose placed directly under the title.
	Intro string

	// Skip lists command names (as they appear in Use, first word) that should
	// be omitted anywhere in the tree. cobra's generated "help" and
	// "completion" commands are always skipped.
	Skip []string

	// IncludeHidden includes commands marked Hidden or Deprecated.
	IncludeHidden bool

	// OmitFlags renders commands without their flag tables.
	OmitFlags bool

	// Compact renders each leaf command as a single bullet line instead of a
	// section with a flag table. Reach for it when the tree is large — a CLI
	// that reflects a whole API into several hundred operations produces a
	// reference no agent can afford to read in the default form.
	Compact bool
}

// Flag is a single command-line flag.
type Flag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage"`
}

// Command is one node of the catalog.
type Command struct {
	Path            string    `json:"path"`
	Use             string    `json:"use"`
	Short           string    `json:"short,omitempty"`
	Long            string    `json:"long,omitempty"`
	Example         string    `json:"example,omitempty"`
	Aliases         []string  `json:"aliases,omitempty"`
	PersistentFlags []Flag    `json:"persistentFlags,omitempty"`
	Flags           []Flag    `json:"flags,omitempty"`
	Commands        []Command `json:"commands,omitempty"`
}

var alwaysSkip = map[string]bool{"help": true, "completion": true}

// Build walks the cobra tree and returns the structured catalog.
func Build(root *cobra.Command, opts Options) Command {
	skip := map[string]bool{}
	for _, s := range opts.Skip {
		skip[s] = true
	}
	return buildCommand(root, root.Name(), opts, skip)
}

func buildCommand(cmd *cobra.Command, path string, opts Options, skip map[string]bool) Command {
	out := Command{
		Path:    path,
		Use:     cmd.Use,
		Short:   cmd.Short,
		Long:    cmd.Long,
		Example: strings.TrimSpace(cmd.Example),
		Aliases: cmd.Aliases,
	}
	if !opts.OmitFlags {
		out.PersistentFlags = collectFlags(cmd.PersistentFlags())
		out.Flags = collectFlags(localFlags(cmd))
	}
	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if alwaysSkip[name] || skip[name] {
			continue
		}
		if !opts.IncludeHidden && (sub.Hidden || sub.Deprecated != "") {
			continue
		}
		out.Commands = append(out.Commands, buildCommand(sub, path+" "+name, opts, skip))
	}
	sort.SliceStable(out.Commands, func(i, j int) bool {
		return out.Commands[i].Path < out.Commands[j].Path
	})
	return out
}

// localFlags returns the flags declared on cmd itself, excluding those it
// inherits from parents (which are reported once at the root).
func localFlags(cmd *cobra.Command) *pflag.FlagSet {
	set := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)
	persistent := cmd.PersistentFlags()
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if persistent.Lookup(f.Name) != nil {
			return
		}
		if cmd.InheritedFlags().Lookup(f.Name) != nil {
			return
		}
		set.AddFlag(f)
	})
	return set
}

func collectFlags(set *pflag.FlagSet) []Flag {
	var flags []Flag
	set.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		flags = append(flags, Flag{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Type:      f.Value.Type(),
			Default:   f.DefValue,
			Usage:     strings.TrimSpace(f.Usage),
		})
	})
	sort.SliceStable(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	return flags
}

// JSON renders the catalog as indented JSON.
func JSON(root *cobra.Command, opts Options) ([]byte, error) {
	return json.MarshalIndent(Build(root, opts), "", "  ")
}

// Markdown renders the catalog as a Markdown chapter suitable for dropping into
// an llmdocs directory.
func Markdown(root *cobra.Command, opts Options) string {
	c := Build(root, opts)

	title := opts.Title
	if title == "" {
		title = "Command catalog"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", title)
	if opts.Intro != "" {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(opts.Intro))
	}

	if len(c.PersistentFlags) > 0 {
		b.WriteString("\n## Global flags\n\n")
		writeFlagTable(&b, c.PersistentFlags)
	}

	if opts.Compact {
		writeCompact(&b, c.Commands, 2)
	} else {
		writeCommands(&b, c.Commands, 2)
	}
	return b.String()
}

// writeCompact renders groups as headings and leaves as one bullet each.
func writeCompact(b *strings.Builder, cmds []Command, depth int) {
	for _, c := range cmds {
		if len(c.Commands) == 0 {
			fmt.Fprintf(b, "- `%s`%s%s\n", usageLine(c), compactFlags(c), compactShort(c))
			continue
		}

		level := depth
		if level > 6 {
			level = 6
		}
		fmt.Fprintf(b, "\n%s `%s`\n", strings.Repeat("#", level), c.Path)
		if c.Short != "" {
			fmt.Fprintf(b, "\n%s\n", c.Short)
		}
		b.WriteString("\n")
		writeCompact(b, c.Commands, depth+1)
	}
}

func compactShort(c Command) string {
	if c.Short == "" {
		return ""
	}
	return " — " + strings.ReplaceAll(c.Short, "\n", " ")
}

func compactFlags(c Command) string {
	if len(c.Flags) == 0 {
		return ""
	}
	names := make([]string, 0, len(c.Flags))
	for _, f := range c.Flags {
		names = append(names, "--"+f.Name)
	}
	return " `[" + strings.Join(names, " ") + "]`"
}

func writeCommands(b *strings.Builder, cmds []Command, depth int) {
	for _, c := range cmds {
		level := depth
		if level > 6 {
			level = 6
		}
		fmt.Fprintf(b, "\n%s `%s`\n", strings.Repeat("#", level), c.Path)

		if c.Short != "" {
			fmt.Fprintf(b, "\n%s\n", c.Short)
		}
		if long := strings.TrimSpace(c.Long); long != "" && long != strings.TrimSpace(c.Short) {
			fmt.Fprintf(b, "\n%s\n", long)
		}
		if usage := usageLine(c); usage != "" {
			fmt.Fprintf(b, "\n```\n%s\n```\n", usage)
		}
		if len(c.Aliases) > 0 {
			fmt.Fprintf(b, "\nAliases: %s\n", strings.Join(c.Aliases, ", "))
		}
		if c.Example != "" {
			fmt.Fprintf(b, "\nExample:\n\n```\n%s\n```\n", c.Example)
		}
		if len(c.Flags) > 0 {
			b.WriteString("\n")
			writeFlagTable(b, c.Flags)
		}
		writeCommands(b, c.Commands, depth+1)
	}
}

// usageLine renders the invocation line, replacing the leaf command name in Use
// with the full command path so an agent can copy it verbatim.
func usageLine(c Command) string {
	use := strings.TrimSpace(c.Use)
	if use == "" {
		return c.Path
	}
	fields := strings.Fields(use)
	if len(fields) == 0 {
		return c.Path
	}
	rest := strings.TrimSpace(strings.TrimPrefix(use, fields[0]))
	if rest == "" {
		return c.Path
	}
	return c.Path + " " + rest
}

func writeFlagTable(b *strings.Builder, flags []Flag) {
	b.WriteString("| flag | type | default | description |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, f := range flags {
		name := "`--" + f.Name + "`"
		if f.Shorthand != "" {
			name = "`-" + f.Shorthand + "`, " + name
		}
		def := f.Default
		if def == "" {
			def = "—"
		} else {
			def = "`" + def + "`"
		}
		usage := strings.ReplaceAll(f.Usage, "|", "\\|")
		usage = strings.ReplaceAll(usage, "\n", " ")
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", name, f.Type, def, usage)
	}
}
