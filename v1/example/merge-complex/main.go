package main

import (
	"fmt"
	"log"
	"os"

	yaml "golang-yaml/v1"
)

func main() {
	fmt.Println("=== YAML Merge With Comments Schema Comments and Doc Comments and Empty Lines ===")

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
  timeout: 30

# @schema
# additionalProperties: false
# @schema
# -- Database settings
database:
  type: postgres
  host: localhost
  port: 5432

# @schema
# additionalProperties: false
# @schema
# -- Features settings list
features:
  - logging
  - metrics
`

	override := `version: 2.0.0
server:
  port: 9000
  ssl: true
database:
  host: db.production.com # This is the production database host
  pool:
    min: 5
    max: 20
features:
  - auth
  - caching
environment: production
`

	fmt.Println("Base YAML:")
	fmt.Println(base)

	fmt.Println("\nOverride YAML:")
	fmt.Println(override)

	mergeOpts := yaml.MergeOptions{
		Mode:               yaml.MergeDeep,
		ArrayMergeStrategy: yaml.ArrayAppend,
		PreserveComments:   true,
	}

	merged, err := yaml.Merge([]byte(base), []byte(override), mergeOpts)
	if err != nil {
		log.Fatalf("Merge failed: %v", err)
	}

	fmt.Println("\nMerged Result (Deep Merge with Array Append):")
	fmt.Println(string(merged))

	// Save the merged result to file
	err = os.WriteFile("yaml-merged-result.yaml", merged, 0644)
	if err != nil {
		log.Fatalf("Failed to save merged result: %v", err)
	}
	fmt.Println("\n✅ Merged result saved to: yaml-merged-result.yaml")

	mergeOptsReplace := yaml.MergeOptions{
		Mode:               yaml.MergeDeep,
		ArrayMergeStrategy: yaml.ArrayReplace,
		PreserveComments:   true,
	}

	mergedReplace, err := yaml.Merge([]byte(base), []byte(override), mergeOptsReplace)
	if err != nil {
		log.Fatalf("Merge failed: %v", err)
	}

	fmt.Println("\nMerged Result (Deep Merge with Array Replace):")
	fmt.Println(string(mergedReplace))

	// Save the array-replaced merged result to file
	err = os.WriteFile("yaml-merged-result-replace.yaml", mergedReplace, 0644)
	if err != nil {
		log.Fatalf("Failed to save merged result: %v", err)
	}
	fmt.Println("\n✅ Array-replace result saved to: yaml-merged-result-replace.yaml")
	fmt.Println()
}