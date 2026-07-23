---
name: example-cli-install
description: Make the example-cli command available, installing it only if it is missing. Use when another skill reports that `example-cli` is not on PATH, or when the user asks to install, update or upgrade example-cli. Prefers an already-installed binary, then the latest GitHub release, then a build from source.
license: MIT
compatibility: Requires curl (or wget) and tar for public releases, an authenticated `gh` CLI for private ones, or a Go toolchain for the source fallback. Standalone — does not need example-cli to be installed already.
allowed-tools: Bash(curl:*) Bash(wget:*) Bash(tar:*) Bash(unzip:*) Bash(gh:*) Bash(go:*) Bash(uname:*) Bash(command:*) Bash(which:*) Bash(mkdir:*) Bash(mv:*) Bash(cp:*) Bash(rm:*) Bash(chmod:*) Bash(ls:*) Bash(test:*) Bash(echo:*) Read
---

# example-cli-install

<!--
TEMPLATE. Copy to plugins/<cli>/skills/<cli>-install/SKILL.md.

Every distributed CLI plugin needs an install skill. Without one, the first
thing a new user hits is "command not found" with no way forward, and that
lands as a support request.

Keep the three routes in order. For a private repository, delete route 2's curl
path — the release assets are not reachable without authentication.
-->

Make `example-cli` usable, doing the least work that achieves it.

## Route 1 — an existing installation on PATH

```bash
command -v example-cli && example-cli --version
```

If that resolves, **use it and stop here.** Do not check for a newer release:
it costs an API call and the user did not ask for an upgrade.

Two things to confirm before trusting the hit:

- **It is the right tool.** Command names collide. Verify the `--version` output
  (or the first line of `example-cli llm`) actually names this project. If a
  different program owns the name, tell the user and use an explicit path to our
  binary instead of shadowing theirs.
- **It is recent enough.** If `example-cli llm` is not recognised, the binary
  predates the agent-facing reference. Say so and continue to route 2 to
  upgrade it.

Continue past this section only when the command is missing, is the wrong tool,
is too old, or the user explicitly asked to update.

## Route 2 — the latest GitHub release

Detect the platform and map it to the release asset naming used by goreleaser
(`Darwin_arm64`, `Linux_x86_64`, …). Check the actual asset names on the release
page before assuming: the casing of the OS and architecture segments follows
that project's `.goreleaser` configuration.

```bash
uname -s   # Darwin | Linux
uname -m   # arm64 | x86_64
```

### Public repository

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/ideamans/example-cli/releases/latest \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)
curl -fsSL -o /tmp/example-cli.tar.gz \
  "https://github.com/ideamans/example-cli/releases/download/${VERSION}/example-cli_$(uname -s)_$(uname -m).tar.gz"
```

### Private repository

Release assets are not reachable anonymously — go through an authenticated
GitHub client:

```bash
gh auth status                                     # confirm authentication first
gh release download --repo ideamans/example-cli \
  --pattern "*$(uname -s)_$(uname -m).tar.gz" --dir /tmp --clobber
```

If `gh` is missing or unauthenticated, or the account lacks access to the
`ideamans` org, **stop and ask the user to grant access.** Give them the exact
command to run themselves:

```
gh auth login
```

Never ask the user to paste a token into the conversation — it would be
recorded in the transcript. `gh auth login` keeps the credential in their
keychain, where it belongs. If they cannot authenticate at all, say what you
need (read access to `ideamans/example-cli`) and stop; do not try to work
around it.

### Install onto PATH

```bash
tar -xzf /tmp/example-cli.tar.gz -C /tmp
mkdir -p ~/.local/bin && mv /tmp/example-cli ~/.local/bin/ && chmod +x ~/.local/bin/example-cli
```

Prefer the first writable directory already on PATH — `~/.local/bin`, then
`/usr/local/bin`. Two things not to do on your own initiative:

- If nothing on PATH is writable, leave the binary in `/tmp`, print the exact
  `sudo mv` command and let the user run it. Do not run `sudo` yourself.
- If `~/.local/bin` is not on PATH, give the line to add to their shell profile.
  Do not edit the profile for them.

## Route 3 — build from source

Last resort: it needs a Go toolchain and compiles rather than downloads. It is
the simplest path when the release assets do not cover the platform, and it
works for private repositories when git credentials are already configured.

```bash
go install github.com/ideamans/example-cli@latest
```

For a private module, `GOPRIVATE=github.com/ideamans/*` must be set and git must
be able to authenticate to GitHub. If it is not, fall back to asking the user as
in route 2.

The binary lands in `$(go env GOPATH)/bin` — check that this is on PATH.

## Verify

```bash
example-cli --version
example-cli llm | head -5
```

Report which route was taken, the version, and the install path — the user needs
to know whether anything was written to their machine. Then continue with what
they originally asked for.
