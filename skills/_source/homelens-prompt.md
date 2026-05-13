# HomeLens — canonical agent prompt source

This is the source of truth for the four per-agent skill files (Claude Code, Codex/Cline/Aider, Cursor, Gemini). When updating skills, edit this file first; the other four mirror it.

## What HomeLens is

Agent-agnostic property search + neighborhood enrichment tool. Given a US city, pulls live Redfin listings, layers in US Census + city-data.com demographics + OSM walkability, computes a within-search Livability score, and renders a single-file HTML report.

CLI binary: `homelens-pp-cli` · MCP server: `homelens-pp-mcp` (4 typed tools)

## When to invoke

Match the user's intent, not exact phrasing. Common triggers:

- "show me properties in <city>" / "homes for sale in <city>"
- "real estate in <city>", "what's for sale in <city> under $X"
- "find me a 3-bed under $600K in Austin"
- "compare <city A> and <city B>"
- "save this search as <name>"
- "watch <saved-name>" / "any new listings since last time?"
- "deep dive on this listing: <redfin URL>"
- "share this report" (uploads as Gist via `gh`)
- "next 25", "more results", "page 2" — continuation of prior search

If the user gives only a city, invoke directly with config defaults — don't ask clarifying questions first.

## How to invoke

```bash
# Default search using user's ~/.config/homelens/config.toml
homelens-pp-cli search "Austin, TX"

# Specific filters override defaults
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# Skip city resolution (faster) if you know the Redfin slug
homelens-pp-cli search "Vancouver, WA" --slug city/18823/WA/Vancouver

# Saved searches
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000
homelens-pp-cli search my-austin                # re-run

# Compare two cities
homelens-pp-cli compare "Austin, TX" "Boise, ID"

# Watch for new listings (exits 9 if anything changed)
homelens-pp-cli watch my-austin

# Single-listing deep-dive (census tract + OSM walkability + amenity counts)
homelens-pp-cli listing https://www.redfin.com/TX/Austin/.../home/12345

# Apply a profile (first-home / investment / downsize / luxury)
homelens-pp-cli search "Salem, OR" --profile first-home

# Output variants
homelens-pp-cli search "Austin, TX" --map           # Leaflet map (CDN-loaded)
homelens-pp-cli search "Austin, TX" --inline-map    # offline-friendly (+160KB)
homelens-pp-cli search "Austin, TX" --md            # Markdown instead of HTML
homelens-pp-cli search "Austin, TX" --pdf           # PDF via headless Chrome
homelens-pp-cli search "Austin, TX" --theme dark    # bloom|modern|classic|minimal|dark

# Share a report as a public Gist (needs `gh` authenticated)
homelens-pp-cli share homelens-austin-tx.html
```

## stdout contract

`search`, `compare`, `listing` print **one line to stdout: the output file path**. All progress messages go to stderr.

After invoking, the agent should:
1. Tell the user the report was generated and give them the path
2. Optionally summarize the top 3-5 matches inline (price, address, livability score)
3. If results exceed `--chunk` (default 25), mention pagination is available

## Exit codes

| Code | Meaning | What to do |
|---:|---|---|
| 0 | ok | proceed |
| 2 | user error (bad flag, invalid city) | check args, ask user for the missing piece |
| 3 | upstream error (Redfin/Census/city-data down) | suggest retry; not the user's fault |
| 4 | rate-limited | wait 60s and retry |
| 5 | auth missing (Census key bad) | tell user to set Census key (optional — only affects deep-dive) |
| 7 | no results | suggest widening filters |
| 9 | changes detected (`watch` only) | surface the new/removed/changed listings to the user |

## Built-in defaults

`min-sqft=1500`, `max-price=$800K`, `min-beds=2`, `min-baths=2`, `types=house+condo+townhouse`, `theme=bloom`, `chunk=25`.

Built-in profiles: `first-home` (≤$450K), `investment` (condos+multi, sort by $/sqft), `downsize` (≤2500sqft), `luxury` (≥$3000sqft).

## Themes

`bloom` (default, pink/lavender mobile-first) · `modern` (navy+gold) · `classic` (serif brochure) · `minimal` (B&W) · `dark` (slate+cyan OLED).

## API keys

**No API keys required to start.** Optional: free Census API key (https://api.census.gov/data/key_signup.html) unlocks tract-level deep-dive demographics. Save via `homelens-pp-cli init` or directly in `~/.config/homelens/config.toml`.

## Companion CLIs (optional but auto-detected)

If installed, HomeLens delegates enrichment to:
- `census-pp-cli` (Geocoder + ACS)
- `city-data-pp-cli` (ZIP scrape)
- `osm-amenities-pp-cli` (walkability)

Falls back to inline implementations otherwise. Either way, behavior is identical from a user's perspective.

## MCP tools (for MCP-aware agents)

`homelens-pp-mcp` exposes 4 typed tools — agents calling via MCP don't need to shell out:

- `search` — full search with enrichment
- `list_searches` — enumerate saved searches
- `listing` — single-listing metadata (lat/lng, year built, description)
- `render_html` — render a prior search result to a themed HTML file
