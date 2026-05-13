// Command homelens-pp-cli is the universal-fallback CLI for HomeLens.
// Any agent that can shell out can use it. The same internal/* packages
// also back homelens-pp-mcp for MCP-aware agents.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	root := &cobra.Command{
		Use:   "homelens-pp-cli",
		Short: "HomeLens — agent-agnostic property search & enrichment",
		Long:  "HomeLens pulls Redfin listings, enriches with Census + city-data demographics, and produces shareable HTML reports. See `homelens-pp-cli agent-context` for the full capability manifest.",
	}
	root.AddCommand(searchCmd())
	root.AddCommand(initCmd())
	root.AddCommand(saveCmd())
	root.AddCommand(listSearchesCmd())
	root.AddCommand(watchCmd())
	root.AddCommand(compareCmd())
	root.AddCommand(listingCmd())
	root.AddCommand(reportCmd())
	root.AddCommand(shareCmd())
	root.AddCommand(profileCmd())
	root.AddCommand(configCmd())
	root.AddCommand(agentContextCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(versionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("homelens-pp-cli", version)
		},
	}
}
