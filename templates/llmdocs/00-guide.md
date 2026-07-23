# <cli> — reference for AI agents

<!--
TEMPLATE. Copy to internal/llmdocs/00-guide.md and replace every <placeholder>.

This chapter is hand-written and is the single source of truth for how an agent
drives the CLI. Keep it factual and imperative: an agent reads it once and then
acts. Chapter order is controlled by the filename prefix, so this file stays at
00-. The command catalog (90-commands.md) is generated — never hand-edit it.
-->

`<cli>` is a non-interactive CLI for <what it wraps>. Every command reads flags
and environment variables only; nothing prompts, so it is safe to run from
scripts and agents. Errors go to stderr prefixed with `Error:`, and the exit
code is 0 on success and non-zero on failure.

This reference is embedded in the binary. `<cli> llm` always describes the exact
version you are running — prefer it over anything you remember about this tool.

## Ground rules

1. Machine-readable output: pass `--json` (or `--format json`) whenever you are
   going to parse the result. The default output is formatted for humans.
2. <Rule that prevents the most common mistake — e.g. never hand-assemble the
   request body; pass the fields as flags and let the CLI build it.>
3. Long or symbol-heavy values go through a file or stdin (`--file -`), not
   through a shell-quoted flag.
4. Destructive commands stop at a confirmation prompt unless `--force` is
   passed. In a non-interactive session, pass it explicitly.
5. <Rule about identifiers: what addresses an object, and where that value comes
   from in earlier output.>

## Authentication

<How credentials are supplied: environment variables, profile files, key files.
Name every variable exactly. State what happens when they are missing, and the
one command that verifies the setup — e.g. `<cli> configure --check`.>

## Typical workflows

### <Workflow name>

```
<cli> <command> --json
<cli> <command> --id "$ID" --json
```

<Two or three sentences on when to use this sequence and what to do with the
output. Prefer one worked example over an exhaustive option list — the option
list is generated in the command catalog chapter.>

## Failure modes

| Symptom | Cause | Fix |
| --- | --- | --- |
| `Error: <message>` | <cause> | <what the agent should do> |

## What this CLI will not do

<Boundaries worth stating so an agent does not attempt them: operations the API
does not expose, actions that require the web console, rate limits.>
