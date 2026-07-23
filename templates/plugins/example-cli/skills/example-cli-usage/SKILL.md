---
name: example-cli-usage
description: Drive the example-cli command-line tool to <do the thing it does>. Use when the user asks to <verb> <object>, mentions example-cli by name, or is working with <the service it wraps>. Loads the CLI's own embedded reference so the flags and subcommands are always those of the installed version.
license: MIT
compatibility: Requires the `example-cli` binary on PATH. Run the example-cli-install skill if it is missing. Credentials come from <ENV_VAR> or a profile created with `example-cli configure`.
allowed-tools: Bash(example-cli:*) Read Write
---

# example-cli-usage

<!--
TEMPLATE. Copy to plugins/<cli>/skills/<cli>-usage/SKILL.md.

Only Agent Skills standard frontmatter belongs here (name, description, license,
compatibility, metadata, allowed-tools) — this file is also installed by
`gh skill install` into Copilot, Cursor and Gemini CLI, which ignore Claude Code
extensions. Claude-specific behaviour goes under metadata.claude-code.*.

The body is a workflow, not a manual. The manual is `example-cli llm`.
-->

## 1. Confirm the tool is available

```bash
command -v example-cli || echo "missing"
```

If it is missing, run the `example-cli-install` skill and come back.

## 2. Load the reference

```bash
example-cli llm
```

This prints the complete, version-accurate reference: conventions,
authentication, the full command catalog and worked examples. Read it before
composing a command — do not guess flags from memory. Use
`example-cli llm --format json` when you want to pick out chapters
programmatically.

## 3. Check credentials

```bash
example-cli <verify-command>
```

<What a healthy result looks like, and what to tell the user when credentials
are missing — which environment variable or profile to set, and where they get
the value.>

## 4. Do the work

Compose the command from the catalog. Rules that matter more than the rest:

- Pass `--json` whenever you are going to parse the output.
- <The one rule agents get wrong most often with this tool.>
- Destructive operations need `--force` in a non-interactive session. Confirm
  with the user in prose first — the flag suppresses the CLI's own prompt.

## 5. Report

Give the user the concrete result — the identifier that was created, the path
that was written, the number of rows returned. On failure, quote the stderr line
verbatim; `Error:`-prefixed messages from this CLI are actionable as written.

## Failure modes

| Symptom | Cause | Fix |
| --- | --- | --- |
| `command not found` | binary not on PATH | run the `example-cli-install` skill |
| `Error: <auth message>` | credentials missing or expired | <the fix> |
| flag rejected as unknown | reference read from memory, not from the binary | re-run `example-cli llm` |
