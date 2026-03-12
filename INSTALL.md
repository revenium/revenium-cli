# Installing the Revenium CLI

## Homebrew (macOS and Linux)

The recommended way to install on macOS and Linux:

```sh
brew install revenium/tap/revenium
```

This installs the binary and sets up shell completions for bash, zsh, and fish automatically.

To upgrade:

```sh
brew upgrade revenium
```

## GitHub Releases (all platforms)

Download pre-built binaries from the [GitHub Releases](https://github.com/revenium/revenium-cli/releases) page.

### macOS

```sh
# Apple Silicon (M1/M2/M3/M4)
curl -Lo revenium.tar.gz https://github.com/revenium/revenium-cli/releases/latest/download/revenium-cli_Darwin_arm64.tar.gz
tar xzf revenium.tar.gz
sudo mv revenium /usr/local/bin/

# Intel
curl -Lo revenium.tar.gz https://github.com/revenium/revenium-cli/releases/latest/download/revenium-cli_Darwin_amd64.tar.gz
tar xzf revenium.tar.gz
sudo mv revenium /usr/local/bin/
```

### Linux

```sh
# x86_64
curl -Lo revenium.tar.gz https://github.com/revenium/revenium-cli/releases/latest/download/revenium-cli_Linux_amd64.tar.gz
tar xzf revenium.tar.gz
sudo mv revenium /usr/local/bin/

# ARM64
curl -Lo revenium.tar.gz https://github.com/revenium/revenium-cli/releases/latest/download/revenium-cli_Linux_arm64.tar.gz
tar xzf revenium.tar.gz
sudo mv revenium /usr/local/bin/
```

### Windows

Download the appropriate zip file from the [releases page](https://github.com/revenium/revenium-cli/releases):

- `revenium-cli_Windows_amd64.zip` (64-bit Intel/AMD)
- `revenium-cli_Windows_arm64.zip` (ARM64)

Extract the zip and add `revenium.exe` to a directory in your `PATH`.

## Go Install

If you have Go installed (1.25+):

```sh
go install github.com/revenium/revenium-cli@latest
```

The binary will be placed in `$GOPATH/bin` (or `$HOME/go/bin` by default).

## Build from Source

```sh
git clone https://github.com/revenium/revenium-cli.git
cd revenium-cli
go build -o revenium .
```

To install into your `$GOPATH/bin`:

```sh
go install .
```

To build with version metadata:

```sh
go build -ldflags "\
  -X github.com/revenium/revenium-cli/internal/build.Version=1.0.0 \
  -X github.com/revenium/revenium-cli/internal/build.Commit=$(git rev-parse HEAD) \
  -X github.com/revenium/revenium-cli/internal/build.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o revenium .
```

## Verifying the Installation

```sh
revenium version
```

You should see output like:

```
revenium 1.0.0 (abc1234)
```

## Shell Completions

If you installed via Homebrew, completions are configured automatically. For manual installations, set up tab completion for your shell:

### Bash

```sh
# Linux
revenium completion bash | sudo tee /etc/bash_completion.d/revenium > /dev/null

# macOS (requires bash-completion@2)
revenium completion bash > $(brew --prefix)/etc/bash_completion.d/revenium
```

### Zsh

```sh
# Ensure completions directory is in fpath (add to ~/.zshrc if needed):
#   autoload -Uz compinit && compinit

revenium completion zsh > "${fpath[1]}/_revenium"
```

### Fish

```sh
revenium completion fish > ~/.config/fish/completions/revenium.fish
```

After setting up completions, restart your shell or source your profile.

## Post-Install Setup

Once installed, configure the CLI with your Revenium credentials:

```sh
# Set your API key (from https://app.revenium.ai)
revenium config set key your-api-key

# Set your team ID (if applicable)
revenium config set team-id your-team-id

# Verify configuration
revenium config show

# Test connectivity
revenium sources list
```

## Uninstalling

### Homebrew

```sh
brew uninstall revenium
brew untap revenium/tap
```

### Manual

Remove the binary and config file:

```sh
rm $(which revenium)
rm -rf ~/.config/revenium
```
