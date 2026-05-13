# HomeLens

Agent-agnostic property search & neighborhood-enrichment tool. Pulls Redfin listings, layers in US Census + city-data.com demographics, computes a within-search **Livability** score, and renders a single-file shareable HTML report.

Built as part of the [printing-press](https://github.com/mvanhorn/cli-printing-press) ecosystem. Works as:

- **CLI** (`homelens-pp-cli`) — universal fallback, any agent can shell out
- **MCP server** (`homelens-pp-mcp`) — first-class integration for MCP-aware agents
- **Skill files** — per-agent installable skills (Claude Code, Codex, Cursor, Gemini)

## Quick start

```bash
go install github.com/ColeMatthewBienek/homelens/cmd/...@latest
homelens-pp-cli search "Vancouver, WA"
```

That opens an HTML report at `homelens-vancouver-wa.html`. Click any card to drill into Redfin.

## Common commands

```bash
# Basic search with config defaults
homelens-pp-cli search "Austin, TX"

# Override filters
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# Save a search to re-run later
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000

# Re-run a saved search
homelens-pp-cli search my-austin

# Switch profile
homelens-pp-cli profile use first-home
homelens-pp-cli search "Salem, OR"

# Inspect config
homelens-pp-cli config show
homelens-pp-cli doctor
```

## Configuration

Configuration lives at `~/.config/homelens/config.toml`. Resolution order:

1. CLI flag (highest priority)
2. `HOMELENS_*` environment variables
3. Active profile (`profile use <name>` or `--profile <name>`)
4. User config (`~/.config/homelens/config.toml`)
5. Built-in defaults (lowest)

Built-in defaults: `min-sqft=1500`, `max-price=$800K`, `min-beds=2`, `min-baths=2`, `types=house+condo+townhouse`, `theme=bloom`, `chunk=25`.

### Profiles

Four profiles ship by default: `first-home`, `investment`, `downsize`, `luxury`. Inspect:

```bash
homelens-pp-cli profile list
```

### Census API key

Tract-level demographics need a free [Census API key](https://api.census.gov/data/key_signup.html). HomeLens auto-detects a key at `~/.config/census-pp-cli/config.toml` (the `census-pp-cli` convention) if present.

## Themes

v0 ships **bloom** (the lovable mobile-first one). Roadmap themes: modern, classic, minimal, dark.

## Roadmap

Status as of v0.1:

| Feature              | Status      | Notes |
|----------------------|-------------|-------|
| `search` w/ Vancouver | ✅ working | reproduces 16-listing baseline |
| bloom theme           | ✅ working   | self-contained HTML |
| Config + profiles    | ✅ working   | 4 built-in profiles |
| Saved searches       | ✅ working   | TOML at `~/.config/homelens/searches/` |
| MCP server           | 🟡 partial | initialize + list_tools work; tool calls shell out to CLI |
| `init` wizard        | 🟡 minimal  | writes defaults; no interactive prompts yet |
| `watch` + diff       | ❌ stub      | next session |
| `compare`            | ❌ stub      | next session |
| `listing` deep dive  | ❌ stub      | next session |
| `share` (gist)       | ❌ stub      | next session |
| PDF / markdown out   | ❌ stub      | next session — needs chromedp |
| Other 4 themes       | ❌ stub      | next session |
| Interactive map      | ❌ stub      | next session — Leaflet inline |

## Architecture

```
cmd/
  homelens-pp-cli/    Cobra CLI (universal fallback)
  homelens-pp-mcp/    MCP server (JSON-RPC over stdio)
internal/
  redfin/             Stingray API client + slug resolver
  census/             Census Geocoder
  citydata/           city-data.com scraper (zip + city)
  score/              Livability composite (within-search percentile)
  render/html/        Themed HTML rendering
  store/              Saved searches + watch snapshots
  config/             TOML + env + profile resolution
```

## Exit codes

| Code | Meaning |
|-----:|---------|
| 0 | ok |
| 2 | user error (bad flag, invalid city) |
| 3 | upstream error (Redfin/Census/city-data down) |
| 4 | rate-limited |
| 5 | auth missing (Census key bad/missing) |
| 7 | no results |
| 9 | changes detected (for `watch` automation) |

## License

MIT
