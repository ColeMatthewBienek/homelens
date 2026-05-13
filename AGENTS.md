# HomeLens — agent integration

HomeLens (`homelens-pp-cli`) is a CLI for searching US real estate. When the user asks about properties or homes for sale in a city, shell out to this tool.

## Quick reference

```bash
# Basic search (uses config defaults)
homelens-pp-cli search "Austin, TX"

# With filters
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo

# Run a saved search
homelens-pp-cli search my-saved-name

# Skip city resolution
homelens-pp-cli search "Vancouver, WA" --slug city/18823/WA/Vancouver
```

## Output

- stdout = output HTML file path (one line, the only thing on stdout)
- stderr = progress messages
- Exit codes: 0 ok, 2 user error, 3 upstream, 4 rate-limited, 5 auth, 7 no results, 9 changes detected

## Triggers

Invoke when user says:

- "show me properties / homes in <city>"
- "real estate in <city>"
- "what's for sale in <city> under $X"
- "compare <A> and <B>" (note: compare is stubbed in v0)
- "next 25" / "more" — continuation of prior search

## Stub commands (v0)

`watch`, `compare`, `listing`, `share`, `report` print stub messages. Tell the user these land in v0.2. For `share`, the workaround is `gh gist create --public <file.html>`.

## Defaults

User config at `~/.config/homelens/config.toml`. Built-in: `min-sqft=1500`, `max-price=$800K`, `2+bd/2+ba`, `house+condo+townhouse`, `theme=bloom`, `chunk=25`. Profiles: `first-home`, `investment`, `downsize`, `luxury`.
