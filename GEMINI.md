# HomeLens — Gemini CLI integration

`homelens-pp-cli` is a CLI for US real-estate research. Use it when the user asks about homes for sale, property values, neighborhoods, or anything similar for a specific US city.

## Trigger phrases

"properties in <city>", "homes for sale", "real estate in", "find me a 3-bed under $X", "compare <A> and <B>", "watch <name>", "deep dive on this listing: <redfin URL>", "next 25", "more results".

When the user gives only a city, invoke directly with their config defaults — don't ask for filters first.

## Invocation

```bash
# Default search using ~/.config/homelens/config.toml
homelens-pp-cli search "Austin, TX"

# Explicit filters
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# Skip city resolution if you know the Redfin slug
homelens-pp-cli search "Vancouver, WA" --slug city/18823/WA/Vancouver

# Saved searches
homelens-pp-cli save my-austin "Austin, TX" --max-price 600000
homelens-pp-cli search my-austin
homelens-pp-cli watch my-austin                  # exits 9 if anything changed

# Compare cities
homelens-pp-cli compare "Austin, TX" "Boise, ID"

# Deep-dive a single listing
homelens-pp-cli listing https://www.redfin.com/...

# Output flags on search
--map           # interactive Leaflet map (small, needs internet)
--inline-map    # offline-friendly map (+160KB)
--md            # Markdown instead of HTML
--pdf           # PDF via headless Chrome
--theme bloom|modern|classic|minimal|dark
--profile first-home|investment|downsize|luxury

# Share
homelens-pp-cli share <html>
```

## Output

- stdout: one line, the output file path
- stderr: progress messages
- Exit codes: 0=ok, 2=user error, 3=upstream, 4=rate-limited, 5=auth missing, 7=no results, 9=changes detected (watch)

## API keys

**None required to start.** Optional free Census key for tract-level deep-dive: https://api.census.gov/data/key_signup.html — save via `homelens-pp-cli init`.

## Defaults

`min-sqft=1500`, `max-price=$800K`, `2+bd/2+ba`, `house+condo+townhouse`, `theme=bloom`. Profiles: `first-home`, `investment`, `downsize`, `luxury`.
