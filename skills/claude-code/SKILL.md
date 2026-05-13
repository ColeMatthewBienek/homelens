---
name: homelens
description: Search for properties in any US city and produce a shareable HTML report with neighborhood demographics and a Livability score. Use when the user says "show me properties in X", "homes for sale", "real estate in Y", or asks about home pricing/neighborhoods in a specific city.
allowed-tools:
  - Bash
---

# HomeLens — property search + neighborhood enrichment

When the user wants to search for homes in a US city, invoke `homelens-pp-cli`. It pulls live Redfin data, enriches with Census + city-data demographics, computes a Livability score, and produces a single-file HTML report.

## Trigger phrases

- "show me properties in <city>"
- "homes for sale in <city> under $X"
- "real estate in <city>"
- "next 25" / "more" (continue prior search)
- "compare <cityA> and <cityB>" (note: compare is stubbed in v0)

## Common usage

```bash
# Default search using user's config defaults
homelens-pp-cli search "Vancouver, WA"

# Explicit filters
homelens-pp-cli search "Austin, TX" --max-price 600000 --min-sqft 1800 --types house,condo,townhouse

# Save a search for later
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000

# Re-run a saved search
homelens-pp-cli search my-austin

# Apply a profile (first-home, investment, downsize, luxury)
homelens-pp-cli search "Boise, ID" --profile first-home

# Skip city resolution (faster if you know the Redfin slug)
homelens-pp-cli search "Vancouver, WA" --slug city/18823/WA/Vancouver
```

## Output contract

- **stdout**: one line, the HTML report file path
- **stderr**: progress messages
- **exit codes**: 0 ok, 2 user error, 3 upstream error, 4 rate-limited, 5 auth missing, 7 no results, 9 changes detected

After running, tell the user the report path and offer to summarize the top matches inline.

## v0 stubs (do not invoke; tell user it's coming)

- `watch`, `compare`, `listing`, `share`, `report` — all stubbed
- Only the `maia` theme is implemented; other themes will land in v0.2

## Defaults

If the user gives only a city, HomeLens uses their `~/.config/homelens/config.toml` defaults (min-sqft=1500, max-price=$800K, 2+bd/2+ba, house+condo+townhouse, theme=maia).
