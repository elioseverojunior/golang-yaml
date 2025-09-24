package main

import (
	"fmt"
	yaml "golang-yaml/v1"
)

func main() {
	yamlStr := `# First comment

# Second comment after blank line
name: test

# Third comment
value: 123`

	fmt.Println("Input:")
	fmt.Println(yamlStr)
	fmt.Println()

	// Parse and marshal back
	node, err := yaml.UnmarshalNode([]byte(yamlStr))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	output, err := yaml.MarshalNode(node)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Output:")
	fmt.Println(string(output))
}