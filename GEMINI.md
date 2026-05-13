# HomeLens — Gemini integration

HomeLens (`homelens-pp-cli`) is a CLI for US real estate search. Use it when the user asks about homes for sale or property values in a specific city.

## Invocation

```bash
homelens-pp-cli search "Austin, TX"
homelens-pp-cli search "Boise, ID" --max-price 500000 --min-sqft 1800 --types house,condo
homelens-pp-cli search my-saved-search-name
```

## Output

- stdout: a single line with the HTML report path
- stderr: progress
- Exit codes: 0=ok, 2=user error, 3=upstream, 4=rate limited, 5=auth missing, 7=no results, 9=changes detected

## Triggers

"properties in <city>", "homes for sale", "real estate in", "compare cities", "next 25".

## v0 stubs

`watch`, `compare`, `listing`, `share`, `report` are not yet implemented — they print informational stubs. Workaround for share: `gh gist create --public <html-file>`.

## Defaults

User config at `~/.config/homelens/config.toml`. Built-ins: min-sqft=1500, max-price=$800K, 2+bd/2+ba, house+condo+townhouse, bloom theme. Profiles available: first-home, investment, downsize, luxury.
