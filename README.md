# go-llm-cli-kit

Make a Go CLI legible to AI agents, from a single source of knowledge.

One directory of Markdown inside your repository becomes four things: the
`<cli> llm` subcommand, a context7 entry, a Claude Code plugin, and an Agent
Skill installable with `gh skill`. This module supplies the Go pieces; `LLM.md`
is the standard the ideamans CLIs follow, and `templates/` holds the files each
CLI copies.

```bash
go get github.com/ideamans/go-llm-cli-kit
```

## Packages

| Package | What it does |
| --- | --- |
| [`llmdocs`](./llmdocs) | Concatenates an embedded directory of Markdown chapters into one reference, as Markdown or JSON |
| [`catalog`](./catalog) | Renders a live cobra command tree as a Markdown or JSON command catalog |
| [`llmcmd`](./llmcmd) | Adds the standard `llm` subcommand, plus backwards compatibility for a legacy `--llm` flag |
| [`skillcheck`](./skillcheck) | Validates a distributable plugin directory: manifest version, Agent Skills frontmatter, discovery keywords |

Only dependency: `spf13/cobra`.

## Shape of a CLI that uses it

```
<cli>/
├── internal/
│   ├── llmdocs/
│   │   ├── llmdocs.go        //go:embed *.md + //go:generate
│   │   ├── 00-guide.md       hand-written
│   │   └── 90-commands.md    generated, committed
│   └── gen-llmdocs/main.go   run by go generate
├── plugins/<cli>/
│   ├── .claude-plugin/plugin.json
│   └── skills/{<cli>-usage,<cli>-install}/SKILL.md
├── context7.json             public repositories only
└── plugin_test.go            runs skillcheck
```

## Wiring

Embed the chapters:

```go
//go:generate go run ../gen-llmdocs
//go:embed *.md
var files embed.FS

func Docs() *kit.Docs { return kit.New(files, ".") }
```

Add the subcommand:

```go
cfg := llmcmd.Config{Docs: llmdocs.Docs()}
llmcmd.AddTo(root, cfg)

// The pre-kit CLIs accepted --llm at any position; keep that working.
if handled, err := llmcmd.HandleLegacy(os.Args[1:], cfg, os.Stdout); handled {
        if err != nil {
                fmt.Fprintln(os.Stderr, "Error:", err)
                os.Exit(1)
        }
        return
}

if err := root.Execute(); err != nil { /* ... */ }
```

Generate the command catalog:

```go
md := catalog.Markdown(cmd.Root(), catalog.Options{
        Title: "Command catalog",
        Skip:  []string{"llm"},
})
os.WriteFile("90-commands.md", []byte(md), 0o644)
```

Guard the plugin:

```go
func TestPluginSkills(t *testing.T) {
        report := skillcheck.CheckDir("plugins/<cli>", skillcheck.Options{
                Version:             version,
                Keywords:            []string{"…"},
                RequireInstallSkill: true,
        })
        for _, p := range report.Problems {
                t.Error(p)
        }
}
```

## What the CLI ends up exposing

```
<cli> llm                   # full reference, Markdown, offline, version-accurate
<cli> llm --format json     # the same chapters as a JSON array
<cli> --llm                 # deprecated alias, still works anywhere on the line
```

## CI

The generated catalog is committed, because `go:embed` needs a real file at
build time. The guard against drift is regenerating it in CI:

```yaml
- run: go generate ./...
- run: git diff --exit-code
- run: go test ./...
```

See `templates/workflows/llm-artifacts.yml`.

## Documentation

- [`LLM.md`](./LLM.md) — the standard: design principles, the four distribution
  channels, public/private differences, the per-repository rollout checklist
- [`templates/`](./templates) — files to copy into a CLI repository

## License

MIT © Ideamans Inc.
