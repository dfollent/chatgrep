# chatgrep

Search your AI coding session history across Claude Code, GitHub Copilot, and Codex from one CLI.

`chatgrep` scans local session files, lets you search by message content, shows surrounding context in an `fzf` preview pane, and prints the exact command to resume the selected session.

## Features

- **Full-text search** across every message in every session, not just titles
- **Fuzzy matching** via `fzf` so you don't need the exact wording
- **Instant results** - no index, no daemon, no database - streams directly from disk
- **Cross-agent** - search Claude Code, Copilot, and Codex from one tool
- **Preview in context** - see surrounding messages before jumping back in
- **One-command resume** - prints the exact `cd` + resume command to pick up where you left off

Built-in history in these tools is limited to recent session titles or basic listing. No full-text search, no fuzzy matching, no way to find that conversation where you solved the auth bug three weeks ago.

## Install

```bash
brew install danfollent/tap/chatgrep
```

Or via `go install`:

```bash
go install github.com/danielfollent/chatgrep/cmd/chatgrep@latest
```

Or build from source:

```bash
git clone https://github.com/danielfollent/chatgrep.git
cd chatgrep
go build -o bin/chatgrep ./cmd/chatgrep
```

Interactive mode requires [`fzf`](https://github.com/junegunn/fzf). Plain mode (`--plain`) does not.

## Quickstart

Browse everything:

```bash
chatgrep
```

Search all providers for a phrase:

```bash
chatgrep "session token"
```

Search only Codex sessions:

```bash
chatgrep --agent codex "deploy staging"
```

Limit results to the current repo:

```bash
chatgrep --project . "auth middleware"
```

Jump straight into the top match:

```bash
eval "$(chatgrep --first "auth middleware")"
```

Use in scripts:

```bash
chatgrep --plain --agent all "rate limit"
```

## CLI

```text
chatgrep [flags] [query]
```

If `query` is omitted, `chatgrep` becomes a session browser.

### Flags

| Flag | Meaning |
| --- | --- |
| `-A`, `--agent` | `claude`, `copilot`, `codex`, or `all` |
| `-p`, `--project` | Filter sessions by working-directory prefix. Use `.` for current directory |
| `--first` | Print resume command for the top match and exit |
| `--plain` | Plain text output; disables `fzf` |
| `--color` | Color output: `auto`, `always`, `never` (default `auto`) |
| `--colors` | Customize colors (repeatable). See [Colors](#colors) |
| `--version` | Print version |

### Query Semantics

- Matching is case-insensitive
- Whitespace splits the query into terms
- All terms must be present in the message text
- Empty query means "show me everything"

## Supported Providers

| Provider | Session source | Resume command |
| --- | --- | --- |
| Claude Code | `~/.claude/projects/*/*.jsonl` | `claude --resume <session-id>` |
| GitHub Copilot CLI | `${COPILOT_CONFIG_DIR:-~/.copilot}/session-state/<session-id>/events.jsonl` | `copilot --resume=<session-id>` |
| Codex | `${CODEX_HOME:-~/.codex}/sessions/**/rollout-*.jsonl` | `codex resume <session-id>` |

Notes:

- In `all` mode, providers with no local session directory are skipped.
- `chatgrep` uses each session's recorded working directory to emit `cd <cwd> && ...` when possible.

## Output Modes

### Interactive

Default when stdout is a TTY.

- sends matches to `fzf`
- preloads the search box with your query
- shows a preview pane with nearby messages
- returns a single resume command on selection

### Plain

Enabled with `--plain` or automatically when piped.

Output format:

```text
provider:sessionID<TAB>timestamp<TAB>role<TAB>snippet
```

Example uses:

```bash
chatgrep --plain "panic" | cut -f1
chatgrep --plain --agent codex "staging deploy" | head
```

## Colors

Color behavior follows the same conventions as `ripgrep`, `grep`, and `git`.

### --color flag

| Value | Behavior |
| --- | --- |
| `auto` | Color when stdout is a TTY (default) |
| `always` | Always emit ANSI colors, even when piped |
| `never` | No ANSI color codes |

### NO_COLOR

Setting the `NO_COLOR` environment variable (any non-empty value) disables colors in `auto` mode. `--color=always` overrides it. See [no-color.org](https://no-color.org).

### Custom colors

Override individual elements with `--colors 'element:attribute:value'`. The flag can be repeated.

```bash
chatgrep --colors 'role.user:fg:red' --colors 'timestamp:style:bold' "query"
```

For persistent customization, set `CHATGREP_COLORS` with semicolon-separated specs:

```bash
export CHATGREP_COLORS='role.user:fg:red;provider.claude:fg:white;timestamp:style:bold'
```

`--colors` flags take precedence over `CHATGREP_COLORS`.

### Elements

| Element | Default | Description |
| --- | --- | --- |
| `provider.claude` | fg:cyan | Claude provider name |
| `provider.copilot` | fg:magenta | Copilot provider name |
| `provider.codex` | fg:yellow | Codex provider name |
| `role.user` | fg:green | User message indicator |
| `role.assistant` | fg:blue | Assistant message indicator |
| `timestamp` | style:dim | Timestamp display |
| `marker` | fg:yellow, style:bold | Preview target marker (`>>>`) |
| `separator` | style:dim | Preview separator line |
| `match` | (none) | Reserved for future use |

### Attributes

- `fg` - foreground color
- `bg` - background color
- `style` - text style: `bold`, `dim`, `underline`, `italic`, `nobold`, `nodim`, `nounderline`, `noitalic`

### Color values

Named colors: `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`

256-color palette: `0`-`255`

## How It Works

- No index, no daemon, no database
- Streams session files directly from disk
- Uses lightweight provider adapters for each agent format
- Keeps Go dependencies at zero; `fzf` is the only external runtime dependency for interactive mode

## Development

Build:

```bash
go build ./cmd/chatgrep
```

Test:

```bash
go test ./...
```
