# HomeLens — agent integration

`homelens-pp-cli` is a CLI for US real-estate research. When the user asks about homes for sale or property values in a US city, shell out to this tool.

## Trigger phrases

- "show me properties in <city>", "homes for sale in <city>", "real estate in <city>"
- "find me a 3-bed under $600K in <city>"
- "compare <A> and <B>"
- "save this search as <name>", "watch <name>"
- "deep dive on this listing: <redfin URL>"
- "next 25" / "more results" — continuation of prior search

When the user gives only a city, invoke directly with their config defaults — don't ask for filters first.

## Quick reference

```bash
# Default search (uses ~/.config/homelens/config.toml)
homelens-pp-cli search "Austin, TX"

# With filters
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# Skip city resolver if you know the slug
homelens-pp-cli search "Vancouver, WA" --slug city/18823/WA/Vancouver

# Saved-search workflow
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000
homelens-pp-cli search my-austin
homelens-pp-cli watch my-austin                  # exits 9 if anything changed

# Compare cities
homelens-pp-cli compare "Austin, TX" "Boise, ID"

# Single-listing deep-dive
homelens-pp-cli listing https://www.redfin.com/...

# Output flags on search
--map | --inline-map | --md | --pdf
--theme bloom|modern|classic|minimal|dark
--profile first-home|investment|downsize|luxury

# Share a report
homelens-pp-cli share <html>
```

## Output

- **stdout**: one line, the output HTML/MD/PDF file path. Nothing else.
- **stderr**: progress messages
- **Exit codes**: `0` ok · `2` user error · `3` upstream · `4` rate-limited · `5` auth missing (Census key — only affects deep-dive) · `7` no results · `9` changes detected (`watch`)

## API keys

**None required to start.** Optional Census key (free, https://api.census.gov/data/key_signup.html) unlocks tract-level deep-dive. Saved via `homelens-pp-cli init` or `~/.config/homelens/config.toml`.

## Themes

`bloom` (default), `modern`, `classic`, `minimal`, `dark`. All produce self-contained HTML.

## Built-in defaults

`min-sqft=1500`, `max-price=$800K`, `2+bd/2+ba`, `house+condo+townhouse`, `chunk=25`. Profiles: `first-home`, `investment`, `downsize`, `luxury`.
