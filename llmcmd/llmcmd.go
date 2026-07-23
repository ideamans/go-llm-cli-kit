// Package llmcmd wires an llmdocs bundle into a cobra CLI as the standard
// `<cli> llm` subcommand.
//
// The standard shape across every ideamans CLI is:
//
//	<cli> llm                      # Markdown reference on stdout
//	<cli> llm --format json        # the same chapters as a JSON array
//	<cli> --llm                    # deprecated alias, kept for compatibility
//
// Wiring it up in main:
//
//	llmcmd.AddTo(root, llmcmd.Config{Docs: llmdocs.New(internalDocs, ".")})
//	if handled, err := llmcmd.HandleLegacy(os.Args[1:], cfg, os.Stdout); handled {
//	        return err
//	}
//
// The legacy scan exists because the pre-kit CLIs exposed --llm as a persistent
// flag that worked at any position on the command line. Keeping the scan means
// `asc apps list --llm` still prints the reference instead of erroring.
package llmcmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ideamans/go-llm-cli-kit/llmdocs"
)

// LegacyFlag is the deprecated whole-command-line flag that predates the llm
// subcommand.
const LegacyFlag = "--llm"

// Config describes the llm subcommand for one CLI.
type Config struct {
	// Docs is the embedded reference. Required.
	Docs *llmdocs.Docs

	// Short overrides the one-line description shown in `<cli> --help`.
	Short string

	// Long overrides the description shown in `<cli> llm --help`.
	Long string
}

func (c Config) short() string {
	if c.Short != "" {
		return c.Short
	}
	return "Print the full reference for AI agents (LLMs)"
}

func (c Config) long() string {
	if c.Long != "" {
		return c.Long
	}
	return "Print the complete, self-contained reference an AI agent needs to drive this CLI:\n" +
		"conventions, authentication, the command catalog and worked examples.\n\n" +
		"The output is embedded in the binary, so it works offline and always matches\n" +
		"the version you are running."
}

// Render returns the reference in the requested format ("markdown", "md" or
// "json").
func Render(cfg Config, format string) (string, error) {
	if cfg.Docs == nil {
		return "", fmt.Errorf("llmcmd: Config.Docs is nil")
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return cfg.Docs.Markdown()
	case "json":
		data, err := cfg.Docs.JSON()
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	default:
		return "", fmt.Errorf("unknown format %q (want markdown or json)", format)
	}
}

// New returns the `llm` subcommand.
func New(cfg Config) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "llm",
		Short: cfg.short(),
		Long:  cfg.long(),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := Render(cfg, format)
			if err != nil {
				return err
			}
			_, err = io.WriteString(cmd.OutOrStdout(), out)
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "markdown", "output format: markdown or json")
	return cmd
}

// AddTo adds the llm subcommand to root along with the deprecated --llm
// persistent flag, and returns the subcommand.
//
// The flag is registered so that `<cli> --llm` parses rather than erroring; the
// actual printing is done by HandleLegacy, which must be called before
// root.Execute so the flag keeps working at any position.
func AddTo(root *cobra.Command, cfg Config) *cobra.Command {
	cmd := New(cfg)
	root.AddCommand(cmd)
	if root.PersistentFlags().Lookup("llm") == nil {
		root.PersistentFlags().Bool("llm", false, "deprecated: use the `llm` subcommand")
		_ = root.PersistentFlags().MarkHidden("llm")
	}
	return cmd
}

// HandleLegacy scans args (normally os.Args[1:]) for the deprecated --llm flag.
// When present it writes the Markdown reference to out and reports true, so the
// caller can return before dispatching to cobra.
//
// Everything after a bare "--" is treated as operands and not scanned.
func HandleLegacy(args []string, cfg Config, out io.Writer) (bool, error) {
	if !hasLegacyFlag(args) {
		return false, nil
	}
	text, err := Render(cfg, "markdown")
	if err != nil {
		return true, err
	}
	_, err = io.WriteString(out, text)
	return true, err
}

func hasLegacyFlag(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == LegacyFlag {
			return true
		}
	}
	return false
}
