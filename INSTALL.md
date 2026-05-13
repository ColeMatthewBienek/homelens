# Installing HomeLens

## Binaries

```bash
go install github.com/ColeMatthewBienek/homelens/cmd/...@latest
```

Produces `homelens-pp-cli` and `homelens-pp-mcp` under `$GOPATH/bin` (typically `~/go/bin`). Add to your PATH if not already.

Verify:

```bash
homelens-pp-cli doctor
```

## Per-agent skill install

HomeLens ships skill files for every major coding agent. Pick the one that matches your agent.

### Claude Code

```bash
# From this repo:
cp skills/claude-code/SKILL.md ~/.claude/skills/homelens/SKILL.md
```

Or, once published to GitHub:

```bash
# (planned, when packaged as a plugin)
gh skill install ColeMatthewBienek/homelens --agent claude-code
```

### Codex CLI / Cline / Aider

Copy `AGENTS.md` (at the repo root) into the project where you want HomeLens to be discoverable. These agents auto-load `AGENTS.md` from the working directory.

### Cursor

Copy `.cursor/rules/homelens.mdc` into your project's `.cursor/rules/` directory. Cursor will auto-load it.

### Gemini CLI

Copy `GEMINI.md` (at the repo root) into your project root.

### Generic MCP (Claude Desktop, Cursor, Cline via MCP, etc.)

Add to your agent's MCP config (e.g. `~/.config/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "homelens": {
      "command": "homelens-pp-mcp"
    }
  }
}
```

Restart the agent, and `homelens` tools (`search`, `list_searches`) become available.

## Census API key (optional but recommended)

For tract-level demographics, get a free key at https://api.census.gov/data/key_signup.html and put it in `~/.config/homelens/config.toml`:

```toml
[census]
api_key = "your-key-here"
```

HomeLens also auto-detects a key at `~/.config/census-pp-cli/config.toml` (printing-press convention).
