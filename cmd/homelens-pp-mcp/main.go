// Command homelens-pp-mcp is the MCP-server wrapper around the HomeLens
// internal packages. v0 is a stub: it implements the JSON-RPC handshake
// and list_tools so MCP-capable agents can discover the surface, but the
// tool calls themselves shell out to homelens-pp-cli for now.
//
// v0.2 will replace the shell-out with direct calls into internal/redfin,
// internal/render/html, etc.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
				"serverInfo":      map[string]any{"name": "homelens-pp-mcp", "version": "0.1.0"},
			}})
			w.Flush()
		case "tools/list":
			enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
				"tools": []map[string]any{
					{
						"name":        "search",
						"description": "Search Redfin for properties in a city with neighborhood enrichment.",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"location":  map[string]string{"type": "string", "description": "City and state, e.g. \"Vancouver, WA\""},
								"max_price": map[string]string{"type": "integer"},
								"min_beds":  map[string]string{"type": "integer"},
								"min_baths": map[string]string{"type": "integer"},
								"min_sqft":  map[string]string{"type": "integer"},
								"types":     map[string]string{"type": "string", "description": "comma-separated: house,condo,townhouse,multi,land"},
								"theme":     map[string]string{"type": "string"},
								"out":       map[string]string{"type": "string"},
							},
							"required": []string{"location"},
						},
					},
					{
						"name":        "list_searches",
						"description": "List saved searches",
						"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
					},
				},
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

func handleToolCall(enc *json.Encoder, w *bufio.Writer, req rpcRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	json.Unmarshal(req.Params, &params)

	args := []string{params.Name}
	if loc, ok := params.Arguments["location"].(string); ok {
		args = append(args, loc)
	}
	for k, v := range params.Arguments {
		if k == "location" {
			continue
		}
		args = append(args, "--"+k, fmt.Sprintf("%v", v))
	}
	out, err := exec.Command("homelens-pp-cli", args...).CombinedOutput()
	if err != nil {
		enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error() + ": " + string(out)}})
	} else {
		enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"content": []map[string]string{{"type": "text", "text": string(out)}},
		}})
	}
	w.Flush()
}
