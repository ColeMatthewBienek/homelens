//go:build ignore
// +build ignore

// scripts/build-skills.go regenerates the 4 per-agent skill files from
// skills/_source/homelens-prompt.md. v0 just copies the prebuilt files;
// v0.2 will template-substitute from the single source.
//
// Run with: go run scripts/build-skills.go
package main

import "fmt"

func main() {
	fmt.Println("Skill files are currently maintained directly. Regeneration template:")
	fmt.Println("  source:       skills/_source/homelens-prompt.md")
	fmt.Println("  targets:")
	fmt.Println("    skills/claude-code/SKILL.md")
	fmt.Println("    AGENTS.md")
	fmt.Println("    .cursor/rules/homelens.mdc")
	fmt.Println("    GEMINI.md")
}
