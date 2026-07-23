# Publishing this plugin

TEMPLATE. Copy to `plugins/<cli>/PUBLISH.md`.

## Before every release

1. `go generate ./...` — regenerate the command catalog, commit any diff.
2. `go test ./...` — includes the `skillcheck` test, which enforces that
   `plugin.json.version` equals the CLI version and that the SKILL.md
   frontmatter stays within the Agent Skills standard.
3. `claude plugin validate plugins/<cli>` — Claude Code's own validator.
4. Bump `plugin.json.version` together with the release tag. The test fails
   otherwise, which is the point: a stale manifest ships silently otherwise.

## Registering in the marketplace (first release only)

Public CLIs go in `ideamans/claude-public-plugins`, private ones in
`ideamans/claude-private-plugins`. Add an entry to `.claude-plugin/marketplace.json`:

```json
{
  "name": "<cli>",
  "source": {
    "source": "git-subdir",
    "url": "https://github.com/ideamans/<cli>.git",
    "path": "plugins/<cli>"
  }
}
```

`git-subdir` points at the default branch of this repository, so subsequent
skill updates reach users as soon as they merge — the marketplace entry itself
only needs touching when the description changes.

## Verifying the published result

```
/plugin marketplace add ideamans/claude-public-plugins
/plugin install <cli>@ideamans-plugins
/<cli>-usage
```

Non-Claude hosts install the same files directly:

```bash
gh skill install ideamans/<cli>/plugins/<cli>/skills/<cli>-usage --agent copilot
gh skill update
```
