package main

import (
	"fmt"
	"log"

	yaml "golang-yaml/v1"
)

func main() {
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
