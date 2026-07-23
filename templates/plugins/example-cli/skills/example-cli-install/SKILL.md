---
name: example-cli-install
description: Install or update the example-cli binary from its GitHub Releases. Use when the user asks to install or upgrade example-cli, or when another skill reports that the `example-cli` command is missing from PATH. Detects OS and architecture, picks the matching release archive, and installs into a writable directory already on PATH.
license: MIT
compatibility: Requires curl (or wget), tar on linux/macos or unzip on windows, and network access to github.com and api.github.com. Standalone — does not need example-cli to be installed already. Private repositories additionally require an authenticated `gh` CLI.
allowed-tools: Bash(curl:*) Bash(wget:*) Bash(tar:*) Bash(unzip:*) Bash(gh:*) Bash(uname:*) Bash(command:*) Bash(which:*) Bash(mkdir:*) Bash(mv:*) Bash(cp:*) Bash(rm:*) Bash(chmod:*) Bash(ls:*) Bash(test:*) Bash(echo:*) Read
---

# example-cli-install

<!--
TEMPLATE. Copy to plugins/<cli>/skills/<cli>-install/SKILL.md.

Every distributed CLI plugin needs an install skill. Without one, the first
thing a new user hits is "command not found" with no way forward, and that
lands as a support request.

For a private repository, drop the curl path entirely and keep only the
`gh release download` path below.
-->

Install or update `example-cli` from the GitHub Releases of
`ideamans/example-cli`.

## 1. Check what is already installed

```bash
command -v example-cli && example-cli --version
```

If the installed version already matches the latest release, stop and say so.

## 2. Detect the platform

```bash
uname -s   # Darwin | Linux
uname -m   # arm64 | x86_64
```

Map to the release asset naming used by goreleaser: `darwin_arm64`,
`darwin_x86_64`, `linux_x86_64`, `linux_arm64`, `windows_x86_64`.

## 3. Download the release

Public repository:

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/ideamans/example-cli/releases/latest | grep '"tag_name"' | head -1 | cut -d'"' -f4)
curl -fsSL -o /tmp/example-cli.tar.gz \
  "https://github.com/ideamans/example-cli/releases/download/${VERSION}/example-cli_${VERSION#v}_$(uname -s)_$(uname -m).tar.gz"
```

Private repository — use the authenticated `gh` CLI instead, and tell the user
to run `gh auth login` if it fails:

```bash
gh release download --repo ideamans/example-cli --pattern '*Darwin_arm64.tar.gz' --dir /tmp --clobber
```

Check the asset names on the release page before assuming the pattern; the
casing of the OS and architecture segments follows that project's `.goreleaser`
configuration.

## 4. Install onto PATH

Unpack, then place the binary in the first writable directory on PATH,
preferring `~/.local/bin`, then `/usr/local/bin`:

```bash
tar -xzf /tmp/example-cli.tar.gz -C /tmp
mkdir -p ~/.local/bin && mv /tmp/example-cli ~/.local/bin/ && chmod +x ~/.local/bin/example-cli
```

If nothing on PATH is writable, leave the binary in `/tmp`, tell the user the
exact `sudo mv` command to finish the job, and do not run `sudo` yourself.

If `~/.local/bin` is not on PATH, say so and give the line to add to their shell
profile — do not edit their profile without asking.

## 5. Verify

```bash
example-cli --version
example-cli llm | head -20
```

Report the installed version and the install path. Then continue with whatever
the user originally asked for.
