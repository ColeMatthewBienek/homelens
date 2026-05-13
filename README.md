# HomeLens

> Ask your agent: **"Show me 3-bed houses in Austin, TX under $600K."**
> 30 seconds later, you have a beautiful, shareable HTML report with neighborhood demographics, a Livability score per ZIP, and an interactive map of every listing.

HomeLens is an agent-agnostic property-search tool. Drop it next to Claude Code, Cursor, Codex, Cline, Aider, Gemini, or any MCP-aware agent — your coding agent becomes your real-estate research assistant. Pulls live Redfin listings, layers in US Census + city-data.com demographics, and renders a single-file HTML report you can share with anyone.

## What you can say to your agent

Once installed, your agent recognizes phrases like:

- *"Show me properties in Vancouver, WA"* → uses your config defaults
- *"Find 3-bed houses in Austin under $600K"*
- *"Real estate in Boise, Idaho, condos and townhouses, at least 1500 sqft"*
- *"Compare Austin and Boise"* → side-by-side two-city report
- *"Save this search as 'my-austin'"* → re-run with `search my-austin`
- *"Watch my-austin"* → diff against last run, surface new listings + price changes
- *"Deep dive on this listing: <redfin URL>"* → census tract + OSM walkability + amenity counts

The agent runs `homelens-pp-cli` under the hood, opens the report, and summarizes the top matches inline.

## Install — 3 steps

### 1. Install Go (one-time)

HomeLens is a single Go binary. If you don't have Go:

- **macOS**: `brew install go`
- **Windows**: `winget install GoLang.Go`
- **Linux**: `sudo apt install golang-go` (or your distro's equivalent)

### 2. Install the HomeLens binaries

```bash
go install github.com/ColeMatthewBienek/homelens/cmd/...@latest
```

This drops `homelens-pp-cli` and `homelens-pp-mcp` into `~/go/bin`. Make sure `~/go/bin` is on your `PATH` (most Go installers handle this).

Verify:

```bash
homelens-pp-cli doctor
```

### 3. Connect your agent

Pick one of the four supported agents (or use the universal MCP route):

<details>
<summary><b>Claude Code</b></summary>

```bash
mkdir -p ~/.claude/skills/homelens
curl -sL https://raw.githubusercontent.com/ColeMatthewBienek/homelens/main/skills/claude-code/SKILL.md \
  -o ~/.claude/skills/homelens/SKILL.md
```

Restart Claude Code. Now say "show me properties in Austin, TX" and it'll invoke HomeLens.
</details>

<details>
<summary><b>Codex CLI / Cline / Aider</b></summary>

These agents auto-load `AGENTS.md` from the working directory:

```bash
curl -sL https://raw.githubusercontent.com/ColeMatthewBienek/homelens/main/AGENTS.md > AGENTS.md
```
</details>

<details>
<summary><b>Cursor</b></summary>

```bash
mkdir -p .cursor/rules
curl -sL https://raw.githubusercontent.com/ColeMatthewBienek/homelens/main/.cursor/rules/homelens.mdc \
  -o .cursor/rules/homelens.mdc
```

Cursor auto-loads it on next open.
</details>

<details>
<summary><b>Gemini CLI</b></summary>

```bash
curl -sL https://raw.githubusercontent.com/ColeMatthewBienek/homelens/main/GEMINI.md > GEMINI.md
```
</details>

<details>
<summary><b>Any MCP-aware host (Claude Desktop, Cursor MCP mode, etc.)</b></summary>

Add to your MCP config (e.g. `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "homelens": {
      "command": "homelens-pp-mcp"
    }
  }
}
```

Restart the host. The HomeLens tools (`search`, `list_searches`, `listing`, `render_html`) become available as native MCP tools.
</details>

That's it. You're done.

## API keys

**Short version: you don't need any API keys to get started.** HomeLens works out of the box with keyless APIs (Redfin Stingray, US Census Geocoder, city-data.com scrape, OSM Overpass).

The one optional key is the **Census Bureau Data API** key, which unlocks tract-level demographics for the listing deep-dive. ZIP-level demographics work without it.

### Getting a free Census API key (optional, ~30 seconds)

1. Visit https://api.census.gov/data/key_signup.html
2. Fill in your name, email, and "Organization" (any value — "personal" works fine)
3. Check your email; the key arrives instantly
4. Save it to your HomeLens config:

```bash
homelens-pp-cli init
# When prompted for "Census API key", paste it in
```

Or edit `~/.config/homelens/config.toml` directly:

```toml
[census]
api_key = "your-40-character-key-here"
```

HomeLens also auto-detects a key at `~/.config/census-pp-cli/config.toml` (printing-press convention) if you already have one.

### What about Walk Score?

HomeLens does **not** use Walk Score's API (free tier requires a domain-matched email; most generic providers are rejected). Instead, the listing deep-dive computes a walkability score from OpenStreetMap Overpass API queries — keyless, no signup.

### What about gh (GitHub CLI)?

Only needed if you want `homelens-pp-cli share <report.html>` to upload a report as a public Gist. Install from https://cli.github.com if you want this feature; otherwise skip.

## Hands-on examples

```bash
# Default search using your config
homelens-pp-cli search "Austin, TX"

# Specific filters override defaults
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# All five themes (re-renders are free)
for t in bloom modern classic minimal dark; do
  homelens-pp-cli search "Vancouver, WA" --theme $t --out van-$t.html
done

# Interactive map (CDN — small file)
homelens-pp-cli search "Austin, TX" --map

# Inline map (160KB heavier, but works offline — great for sharing)
homelens-pp-cli search "Austin, TX" --inline-map

# PDF (requires Chrome or Edge installed)
homelens-pp-cli search "Austin, TX" --pdf

# Markdown report
homelens-pp-cli search "Austin, TX" --md

# Save & watch a search for new listings
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000
homelens-pp-cli watch my-austin     # exits with code 9 if anything changed

# Compare two cities
homelens-pp-cli compare "Austin, TX" "Boise, ID"

# Deep-dive a single listing (census tract + walkability + amenities)
homelens-pp-cli listing https://www.redfin.com/TX/Austin/...

# Apply a profile
homelens-pp-cli profile use first-home   # built-ins: first-home, investment, downsize, luxury
homelens-pp-cli search "Salem, OR"
```

## Configuration

Lives at `~/.config/homelens/config.toml`. Resolution order:

1. CLI flag (highest)
2. `HOMELENS_*` environment variable
3. Active profile
4. User config
5. Built-in defaults (lowest)

Built-in defaults: `min-sqft=1500`, `max-price=$800K`, `min-beds=2`, `min-baths=2`, `types=house+condo+townhouse`, `theme=bloom`, `chunk=25`.

Built-in profiles: `first-home`, `investment`, `downsize`, `luxury`. Inspect with `homelens-pp-cli profile list`.

Run `homelens-pp-cli init` for an interactive walkthrough.

## Themes

Five ship — same data, different aesthetics:

- **bloom** — pink/lavender, mobile-first, friendly (default — great for sharing with non-technical friends)
- **modern** — navy + gold, professional
- **classic** — Georgia serif, brochure-style
- **minimal** — B&W Tufte
- **dark** — slate + cyan, OLED-friendly

## CLI command status

| Command | Status |
|---|---|
| `search` | ✅ 5 themes, `--map`, `--inline-map`, `--md`, `--pdf` |
| `save` / `list-searches` | ✅ |
| `watch <name>` | ✅ diff vs last run, exit 9 on changes |
| `compare <a> <b>` | ✅ side-by-side two-city report |
| `listing <url>` | ✅ census tract + OSM amenities + walkability score |
| `share <html>` | ✅ wraps `gh gist create` |
| `init` | ✅ interactive wizard |
| `profile list/use` | ✅ |
| `config show/edit` | ✅ |
| `doctor` | ✅ |
| `agent-context` | ✅ JSON capability manifest |
| `report` | ❌ deferred — re-run `search` with `--theme X` to re-render for now |
| Standalone enrichment CLIs | ✅ — `census-pp-cli`, `city-data-pp-cli`, `osm-amenities-pp-cli` ship as printing-press library entries. HomeLens prefers them when on PATH and falls back to inline when not. |

## Architecture

```
cmd/
  homelens-pp-cli/    Cobra CLI (universal fallback)
  homelens-pp-mcp/    MCP server (JSON-RPC over stdio, direct internal calls)
internal/
  redfin/             Stingray API client + slug resolver
  census/             Census Geocoder
  citydata/           city-data.com scraper
  osm/                OSM Overpass + walkability composite
  listing/            Single-listing deep-dive (HTML + HTML-scrape fetcher)
  score/              Livability composite (within-search percentile)
  compare/            Two-city side-by-side renderer
  diff/               Watch snapshot diff
  mapview/            Leaflet HTML snippet builder (CDN + inline modes)
  render/html/        5 themed HTML templates
  render/md/          Markdown renderer
  render/pdf/         chromedp / headless Chrome PDF
  store/              Saved searches + watch history
  share/              `gh gist create` wrapper
  config/             TOML + env + profile resolution
```

## Exit codes

| Code | Meaning |
|-----:|---------|
| 0 | ok |
| 2 | user error (bad flag, invalid city) |
| 3 | upstream error (Redfin/Census/city-data down) |
| 4 | rate-limited |
| 5 | auth missing (Census key bad/missing — only matters for tract-level deep-dive) |
| 7 | no results |
| 9 | changes detected (for `watch` automation) |

## Data sources & attribution

- **Listings** — Redfin Stingray API (public, keyless, rate-limited)
- **ZIP demographics** — [city-data.com](https://www.city-data.com) (HTML scrape, via `city-data-pp-cli` if installed)
- **Census tract & FIPS** — [US Census Bureau Geocoder](https://geocoding.geo.census.gov) (keyless, via `census-pp-cli` if installed)
- **Tract demographics** — [US Census Bureau Data API](https://www.census.gov/data/developers.html) (free key, via `census-pp-cli`)
- **Amenities & walkability** — [OpenStreetMap Overpass API](https://overpass-api.de) (keyless, via `osm-amenities-pp-cli` if installed)
- **Maps** — [Leaflet](https://leafletjs.com) + [OpenStreetMap tiles](https://www.openstreetmap.org)

## Printing-press companions

HomeLens calls out to three companion CLIs from the [printing-press](https://github.com/mvanhorn/cli-printing-press) library. Installing them lets HomeLens (and any other tool you build) share one cached enrichment layer:

| CLI | What it does | Auth |
|---|---|---|
| `census-pp-cli` | US Census Geocoder + ACS 5-year demographics | free key for ACS |
| `city-data-pp-cli` | city-data.com ZIP/city scraper | none |
| `osm-amenities-pp-cli` | OpenStreetMap amenity counts + walkability composite | none |

Build each from `~/printing-press/library/<name>/`. HomeLens auto-detects them on PATH and delegates; if they're missing, HomeLens uses inline equivalents (zero install friction for standalone use).

## License

MIT
