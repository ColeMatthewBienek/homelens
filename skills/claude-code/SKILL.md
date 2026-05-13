---
name: homelens
description: Search for properties in any US city and produce a shareable HTML report with neighborhood demographics, a Livability score, and an interactive map. Invoke when the user says "show me properties in X", "homes for sale", "real estate in", "compare cities", "next 25", or pastes a Redfin listing URL for a deep-dive.
allowed-tools:
  - Bash
---

# HomeLens — property search + neighborhood enrichment

Agent-agnostic CLI for US real estate research. Pulls live Redfin listings, layers in US Census + city-data.com demographics + OSM walkability, and renders a single-file HTML report.

## Trigger phrases

Match intent, not exact wording. Common triggers:

- "show me properties in <city>" / "homes for sale in <city>"
- "real estate in <city>", "what's for sale under $X"
- "find me a 3-bed under $600K in Austin"
- "compare <A> and <B>"
- "save this search as <name>"
- "watch <name>" / "any new listings?"
- "deep dive on this listing: <redfin URL>"
- "share this report"
- "next 25", "more", "page 2" — continuation

When the user gives only a city, invoke directly with their config defaults — don't ask for filters.

## How to invoke

```bash
# Default search (uses ~/.config/homelens/config.toml)
homelens-pp-cli search "Austin, TX"

# Explicit filters
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# Skip city resolution if you know the Redfin slug (faster)
homelens-pp-cli search "Vancouver, WA" --slug city/18823/WA/Vancouver

# Save / re-run / watch
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000
homelens-pp-cli search my-austin
homelens-pp-cli watch my-austin                  # exits 9 if anything changed

# Compare two cities
homelens-pp-cli compare "Austin, TX" "Boise, ID"

# Deep-dive a single listing (census tract + OSM walkability + amenities)
homelens-pp-cli listing https://www.redfin.com/TX/Austin/.../home/12345

# Profiles
homelens-pp-cli search "Salem, OR" --profile first-home   # first-home|investment|downsize|luxury

# Output variants on search
homelens-pp-cli search "Austin, TX" --map                 # interactive Leaflet map (CDN)
homelens-pp-cli search "Austin, TX" --inline-map          # fully offline (+160KB)
homelens-pp-cli search "Austin, TX" --md                  # Markdown instead of HTML
homelens-pp-cli search "Austin, TX" --pdf                 # PDF via headless Chrome
homelens-pp-cli search "Austin, TX" --theme dark          # bloom|modern|classic|minimal|dark

# Share as public Gist (requires `gh` authenticated)
homelens-pp-cli share homelens-austin-tx.html
```

## Output contract

`search` / `compare` / `listing` print **one line on stdout**: the output file path. Progress goes to stderr.

After running, tell the user the path and offer to summarize the top matches inline.

## Exit codes

`0` ok · `2` user error · `3` upstream error · `4` rate-limited · `5` Census key missing (only affects deep-dive) · `7` no results · `9` changes detected (`watch`)

## API keys

**None required to start.** Optional: free Census key from https://api.census.gov/data/key_signup.html unlocks tract-level deep-dive. Run `homelens-pp-cli init` to save it.

## Built-in defaults

`min-sqft=1500`, `max-price=$800K`, `min-beds=2`, `min-baths=2`, `types=house+condo+townhouse`, `theme=bloom`.
