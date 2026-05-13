// Command homelens-pp-mcp is the MCP-server wrapper around HomeLens.
// v0.3 calls internal packages directly instead of shelling out to the CLI.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/ColeMatthewBienek/homelens/internal/citydata"
	"github.com/ColeMatthewBienek/homelens/internal/config"
	"github.com/ColeMatthewBienek/homelens/internal/listing"
	"github.com/ColeMatthewBienek/homelens/internal/redfin"
	htmlrender "github.com/ColeMatthewBienek/homelens/internal/render/html"
	"github.com/ColeMatthewBienek/homelens/internal/score"
	"github.com/ColeMatthewBienek/homelens/internal/store"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	enc := json.NewEncoder(w)

	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		switch req.Method {
		case "initialize":
			enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "homelens-pp-mcp", "version": "0.3.0"},
			}})
			w.Flush()
		case "tools/list":
			enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
				"tools": tools(),
			}})
			w.Flush()
		case "tools/call":
			handleToolCall(enc, w, req)
		default:
			enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found"}})
			w.Flush()
		}
	}
}

func tools() []map[string]any {
	return []map[string]any{
		{
			"name":        "search",
			"description": "Search Redfin for properties in a city with neighborhood enrichment. Returns a structured JSON result with listings + ZIP demographics + livability scores.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location":  map[string]any{"type": "string", "description": "City and state, e.g. \"Vancouver, WA\""},
					"slug":      map[string]any{"type": "string", "description": "Redfin region slug like city/18823/WA/Vancouver (skip city resolution)"},
					"max_price": map[string]any{"type": "integer"},
					"min_beds":  map[string]any{"type": "integer"},
					"min_baths": map[string]any{"type": "integer"},
					"min_sqft":  map[string]any{"type": "integer"},
					"types":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Subset of: house, condo, townhouse, multi, land"},
					"no_enrich": map[string]any{"type": "boolean", "description": "Skip city-data ZIP demographics enrichment (faster)"},
				},
				"required": []string{"location"},
			},
		},
		{
			"name":        "list_searches",
			"description": "List saved searches stored under ~/.config/homelens/searches/",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "listing",
			"description": "Fetch a single Redfin listing's metadata (year built, lot size, description, lat/lng)",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]any{"type": "string"}},
				"required":   []string{"url"},
			},
		},
		{
			"name":        "render_html",
			"description": "Render a previously fetched search result set to a themed HTML report.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location":   map[string]any{"type": "string"},
					"homes_json": map[string]any{"type": "string", "description": "JSON-encoded array of redfin.Home (from a prior `search` call)"},
					"theme":      map[string]any{"type": "string", "description": "bloom | modern | classic | minimal | dark"},
					"out":        map[string]any{"type": "string", "description": "output file path"},
				},
				"required": []string{"location", "homes_json", "out"},
			},
		},
	}
}

func handleToolCall(enc *json.Encoder, w *bufio.Writer, req rpcRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	json.Unmarshal(req.Params, &params)

	result, err := callTool(params.Name, params.Arguments)
	if err != nil {
		enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}})
		w.Flush()
		return
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
		"content": []map[string]string{{"type": "text", "text": string(b)}},
	}})
	w.Flush()
}

func callTool(name string, args map[string]any) (any, error) {
	switch name {
	case "search":
		return mcpSearch(args)
	case "list_searches":
		return mcpListSearches()
	case "listing":
		url, _ := args["url"].(string)
		return listing.Fetch(url)
	case "render_html":
		return mcpRenderHTML(args)
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

func mcpSearch(args map[string]any) (any, error) {
	cfg, _ := config.Load()
	defaults := cfg.Defaults

	loc, _ := args["location"].(string)
	slug, _ := args["slug"].(string)
	noEnrich, _ := args["no_enrich"].(bool)

	getInt := func(k string, fallback int) int {
		if v, ok := args[k].(float64); ok && v > 0 {
			return int(v)
		}
		return fallback
	}
	types := defaults.Types
	if t, ok := args["types"].([]any); ok && len(t) > 0 {
		types = nil
		for _, x := range t {
			if s, ok := x.(string); ok {
				types = append(types, s)
			}
		}
	}

	if slug == "" {
		s, err := redfin.ResolveCity(loc)
		if err != nil {
			return nil, err
		}
		slug = s
	}
	homes, err := redfin.Search(slug, redfin.Filters{
		MaxPrice: getInt("max_price", defaults.MaxPrice),
		MinBeds:  getInt("min_beds", defaults.MinBeds),
		MinBaths: getInt("min_baths", defaults.MinBaths),
		MinSqFt:  getInt("min_sqft", defaults.MinSqFt),
		Types:    config.TypesToUIPT(types),
		Status:   1,
	}, 3)
	if err != nil {
		return nil, err
	}

	zips := map[string]*citydata.ZipProfile{}
	zipCity := map[string]string{}
	if !noEnrich {
		unique := map[string]bool{}
		for _, h := range homes {
			if h.Zip != "" {
				unique[h.Zip] = true
				if _, ok := zipCity[h.Zip]; !ok {
					zipCity[h.Zip] = h.City
				}
			}
		}
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		for z := range unique {
			wg.Add(1)
			sem <- struct{}{}
			go func(z string) {
				defer wg.Done()
				defer func() { <-sem }()
				if p, err := citydata.FetchZip(z); err == nil {
					mu.Lock()
					zips[z] = p
					mu.Unlock()
				}
			}(z)
		}
		wg.Wait()
	}

	livability := map[string]int{}
	for z := range zips {
		livability[z] = score.Livability(z, zips)
	}
	sortedZips := make([]string, 0, len(zips))
	for z := range zips {
		sortedZips = append(sortedZips, z)
	}
	sort.Slice(sortedZips, func(i, j int) bool {
		return livability[sortedZips[i]] > livability[sortedZips[j]]
	})

	return map[string]any{
		"location":    loc,
		"slug":        slug,
		"count":       len(homes),
		"homes":       homes,
		"zips":        zips,
		"zip_city":    zipCity,
		"livability":  livability,
		"sorted_zips": sortedZips,
	}, nil
}

func mcpListSearches() (any, error) {
	names, err := store.ListSearches()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(names))
	for _, n := range names {
		s, err := store.LoadSearch(n)
		if err != nil {
			continue
		}
		out = append(out, map[string]string{"name": n, "location": s.Location})
	}
	return out, nil
}

func mcpRenderHTML(args map[string]any) (any, error) {
	loc, _ := args["location"].(string)
	homesJSON, _ := args["homes_json"].(string)
	theme, _ := args["theme"].(string)
	out, _ := args["out"].(string)
	if theme == "" {
		theme = "bloom"
	}
	var homes []redfin.Home
	if err := json.Unmarshal([]byte(homesJSON), &homes); err != nil {
		return nil, fmt.Errorf("homes_json parse: %w", err)
	}
	f, err := os.Create(out)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := htmlrender.Render(theme, htmlrender.Data{
		Location: loc,
		Homes:    homes,
	}, f); err != nil {
		return nil, err
	}
	return map[string]string{"path": out}, nil
}
