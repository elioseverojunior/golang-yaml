package main

import (
	"fmt"
	"log"

	"golang-yaml/v1"
)

func demonstrateMerge() {
	fmt.Println("\n=== YAML Merge Functionality Demo ===")

	base := `
name: MyApp
version: 1.0.0
server:
  host: localhost
  port: 8080
  timeout: 30
database:
  type: postgres
  host: localhost
  port: 5432
features:
  - logging
  - metrics
`

	override := `
version: 2.0.0
server:
  port: 9000
  ssl: true
database:
  host: db.production.com
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
}

func demonstrateCommentPreservation() {
	fmt.Println("\n=== Comment Preservation Demo ===")

	yamlWithComments := `# Application configuration
name: MyApp # The application name
version: 1.0.0

# Server settings
server:
  host: localhost # Server hostname
  port: 8080 # Server port

# Database configuration
database:
  type: postgres # Database type
  host: localhost
  port: 5432
`

	fmt.Println("Original YAML with comments:")
	fmt.Println(yamlWithComments)

	node, err := yaml.UnmarshalNode([]byte(yamlWithComments))
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	output, err := yaml.MarshalNode(node)
	if err != nil {
		log.Fatalf("Failed to marshal YAML: %v", err)
	}

	fmt.Println("\nRe-marshaled YAML (comments preserved):")
	fmt.Println(string(output))
	fmt.Println()
}

func demonstrateSorting() {
	fmt.Println("\n=== Node Sorting Demo ===")

	unsortedYAML := `
zebra: last
apple: first
middle: center
banana: second
`

	node, err := yaml.UnmarshalNode([]byte(unsortedYAML))
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	doc := node.(*yaml.Document)
	if len(doc.Content) > 0 {
		if mapping, ok := doc.Content[0].(*yaml.Mapping); ok {
			mapping.Sort(yaml.SortAscending, yaml.SortKeys, nil)
		}
	}

	sorted, err := yaml.MarshalNode(node)
	if err != nil {
		log.Fatalf("Failed to marshal sorted YAML: %v", err)
	}

	fmt.Println("Sorted YAML (by keys, ascending):")
	fmt.Println(string(sorted))
}

func demonstrateYAML12Features() {
	fmt.Println("\n=== YAML 1.2.2 Features Demo ===")

	yamlFeatures := `
# Anchors and Aliases
defaults: &defaults
  timeout: 30
  retries: 3

service1:
  <<: *defaults
  port: 8080

service2:
  <<: *defaults
  port: 9090

# Different scalar styles
plain: This is a plain scalar
single: 'This is a single-quoted scalar'
double: "This is a double-quoted scalar with\nescape sequences"
literal: |
  This is a literal block scalar.
  Line breaks are preserved.
    Indentation too.
folded: >
  This is a folded block scalar.
  Line breaks are converted
  to spaces unless they are empty.

# Number formats
decimal: 12345
octal: 0o14
hex: 0xC
binary: 0b1010
float: 3.14159
infinity: .inf
not_a_number: .nan

# Boolean values
yes_values: [true, yes, on]
no_values: [false, no, off]

# Null values
nulls: [null, ~]

# Flow collections
flow_seq: [1, 2, 3, 4, 5]
flow_map: {key1: value1, key2: value2}
`

	fmt.Println("YAML 1.2.2 Features:")
	fmt.Println(yamlFeatures)

	var data interface{}
	err := yaml.Unmarshal([]byte(yamlFeatures), &data)
	if err != nil {
		log.Fatalf("Failed to unmarshal: %v", err)
	}

	marshaled, err := yaml.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to marshal: %v", err)
	}

	fmt.Println("\nRound-trip result:")
	fmt.Println(string(marshaled))
}

func main() {
	fmt.Println("=== golang-yaml Library Demo ===")
	fmt.Println("A comprehensive YAML 1.2.2 implementation with advanced features")

	demonstrateMerge()
	demonstrateCommentPreservation()
	demonstrateSorting()
	demonstrateYAML12Features()

	fmt.Println("\n=== Demo Complete ===")
}
