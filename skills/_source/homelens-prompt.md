# HomeLens prompting source

This is the canonical agent prompt for HomeLens. The four per-agent skill files (Claude Code, Codex, Cursor, Gemini) are generated from this single source via `scripts/build-skills.go`.

## What HomeLens is

A property search + neighborhood enrichment tool. Given a US city, it pulls live Redfin listings, enriches each ZIP with US Census + city-data.com demographics, computes a within-search Livability score, and renders a shareable HTML report.

## When to invoke

- User says "show me properties in <city>"
- User asks "what homes are for sale in <city> under $X?"
- User asks about a neighborhood's demographics, walkability, or livability in the context of a home search
- User says "next 25" or "more results" — treat as continuation of prior search
- User mentions comparing two cities for home prices/livability
- User asks to deep-dive on a specific Redfin listing URL

## How to invoke

The CLI is `homelens-pp-cli`. Always use absolute filter values from the user when they specify; otherwise the CLI applies the user's config defaults from `~/.config/homelens/config.toml`.

### Common one-liners

```bash
# Default search
homelens-pp-cli search "Austin, TX"

# Specific filters
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# Apply a profile
homelens-pp-cli search "Salem, OR" --profile first-home

# Save a search
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000

# Re-run a saved search
homelens-pp-cli search my-austin

# Skip the city resolver (faster if you know the slug)
homelens-pp-cli search "Vancouver, WA" --slug city/18823/WA/Vancouver
```

## Output handling

`homelens-pp-cli search` prints **the output HTML file path** to stdout (one line). All progress messages go to stderr.

When the user is in a chat context, after running the CLI:
1. Tell the user the report was generated and give them the path
2. Offer to summarize the top N matches inline
3. If the report has > 25 matches, mention pagination and offer "next 25"

## Error handling

| Exit code | Meaning | What to do |
|---:|---|---|
| 2 | user error | Re-explain the args or ask user for missing piece |
| 3 | upstream error | Suggest retry; Redfin/Census/city-data may be down |
| 4 | rate-limited | Wait 60s and retry |
| 5 | auth missing | Tell user to set Census key at `~/.config/homelens/config.toml` |
| 7 | no results | Suggest widening filters |
| 9 | changes detected | Surface the diff to the user |

## Stub features in v0

Tell the user these are coming in v0.2; do not try to invoke them:

- `compare` (side-by-side city comparison)
- `watch` (diff-against-last-run)
- `listing` (single-listing deep dive)
- `share` (gist upload — workaround: `gh gist create --public <file>`)
- `report` (PDF / markdown export)

## Defaults a new user gets

`min-sqft=1500`, `max-price=$800K`, `min-beds=2`, `min-baths=2`, `types=house+condo+townhouse`, `theme=maia`, chunk=25.

Built-in profiles: `first-home`, `investment`, `downsize`, `luxury`.
