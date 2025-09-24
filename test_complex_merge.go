package main

import (
	"fmt"
	yaml "golang-yaml/v1"
)

func main() {
	base := `# yaml-language-server: $schema=values.schema.json
# Default values for base-chart.
# This is a YAML-formatted file.

# Declare variables to be passed into your templates.

# @schema
# additionalProperties: false
# @schema
# -- Application configuration
name: MyApp # The application name

# @schema
# additionalProperties: false
# @schema
# -- Application Version
version: 1.0.0

# @schema
# additionalProperties: false
# @schema
# -- Server settings
server:
  host: localhost
  port: 8080
  timeout: 30`

	override := `version: 2.0.0
server:
  port: 9000
  ssl: true`

	fmt.Println("Testing complex merge with schema comments and blank lines...")
	fmt.Println("\nBase has:")
	fmt.Println("- Schema comments (# @schema)")
	fmt.Println("- Documentation comments (# --)")
	fmt.Println("- Inline comments")
	fmt.Println("- Blank lines")

	mergeOpts := yaml.MergeOptions{
		Mode:               yaml.MergeDeep,
		ArrayMergeStrategy: yaml.ArrayReplace,
		PreserveComments:   true,
	}

	merged, err := yaml.Merge([]byte(base), []byte(override), mergeOpts)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Println("\n=== MERGED RESULT ===")
	fmt.Println(string(merged))
	fmt.Println("=== END RESULT ===")

	// Check what was preserved
	result := string(merged)

	fmt.Println("\nPreservation Check:")
	if contains(result, "# yaml-language-server:") {
		fmt.Println("✅ Language server comment preserved")
	} else {
		fmt.Println("❌ Language server comment LOST")
	}

	if contains(result, "# @schema") {
		fmt.Println("✅ Schema comments preserved")
	} else {
		fmt.Println("❌ Schema comments LOST")
	}

	if contains(result, "# The application name") {
		fmt.Println("✅ Inline comments preserved")
	} else {
		fmt.Println("❌ Inline comments LOST")
	}

	// Count blank lines
	blankCount := 0
	for _, line := range []byte(result) {
		if line == '\n' {
			blankCount++
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}