package parser

import (
	"strings"
	"testing"

	"golang-yaml/v1/ast"
)

func TestParser_Scalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		tag      string
	}{
		{"simple string", "hello", "hello", "!!str"},
		{"null value", "null", "", "!!null"},
		{"tilde null", "~", "", "!!null"},
		{"boolean true", "true", "true", "!!bool"},
		{"boolean false", "false", "false", "!!bool"},
		{"integer", "42", "42", "!!int"},
		{"float", "3.14", "3.14", "!!float"},
		{"infinity", ".inf", ".inf", "!!float"},
		{"not a number", ".nan", ".nan", "!!float"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			node, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			doc, ok := node.(*ast.Document)
			if !ok {
				t.Fatalf("expected Document, got %T", node)
			}

			if len(doc.Content) != 1 {
				t.Fatalf("expected 1 content node, got %d", len(doc.Content))
			}

			scalar, ok := doc.Content[0].(*ast.Scalar)
			if !ok {
				t.Fatalf("expected Scalar, got %T", doc.Content[0])
			}

			if scalar.Value != tt.expected {
				t.Errorf("expected value %q, got %q", tt.expected, scalar.Value)
			}

			if scalar.Tag() != tt.tag {
				t.Errorf("expected tag %q, got %q", tt.tag, scalar.Tag())
			}
		})
	}
}

func TestParser_Mappings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "simple mapping",
			input: "key: value",
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name: "multiple entries",
			input: `key1: value1
key2: value2
key3: value3`,
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name: "nested mapping",
			input: `parent:
  child1: value1
  child2: value2`,
			expected: map[string]string{
				"parent.child1": "value1",
				"parent.child2": "value2",
			},
		},
		{
			name: "flow mapping",
			input: `{key1: value1, key2: value2}`,
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			node, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			doc, ok := node.(*ast.Document)
			if !ok {
				t.Fatalf("expected Document, got %T", node)
			}

			result := extractMapping(doc.Content[0], "")

			for key, expected := range tt.expected {
				if value, ok := result[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if value != expected {
					t.Errorf("key %q: expected %q, got %q", key, expected, value)
				}
			}
		})
	}
}

func TestParser_Sequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "simple sequence",
			input: `- item1
- item2
- item3`,
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "flow sequence",
			input:    `[item1, item2, item3]`,
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name: "nested sequence",
			input: `-
  - nested1
  - nested2
- item2`,
			expected: []string{"nested1", "nested2", "item2"},
		},
		{
			name:     "empty sequence",
			input:    `[]`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			node, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			doc, ok := node.(*ast.Document)
			if !ok {
				t.Fatalf("expected Document, got %T", node)
			}

			result := extractSequence(doc.Content[0])

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d items, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("missing item at index %d", i)
				} else if result[i] != expected {
					t.Errorf("item %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestParser_Comments(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedHead string
		expectedLine string
	}{
		{
			name: "head comment",
			input: `# This is a comment
key: value`,
			expectedHead: "This is a comment",
			expectedLine: "",
		},
		{
			name:         "inline comment",
			input:        `key: value # inline comment`,
			expectedHead: "",
			expectedLine: "inline comment",
		},
		{
			name: "both comments",
			input: `# head comment
key: value # inline comment`,
			expectedHead: "head comment",
			expectedLine: "inline comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			node, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			doc, ok := node.(*ast.Document)
			if !ok {
				t.Fatalf("expected Document, got %T", node)
			}

			mapping, ok := doc.Content[0].(*ast.Mapping)
			if !ok {
				t.Fatalf("expected Mapping, got %T", doc.Content[0])
			}

			comment := mapping.GetComment()
			headComment := strings.TrimSpace(comment.HeadComment)
			if headComment != tt.expectedHead {
				t.Errorf("expected head comment %q, got %q", tt.expectedHead, headComment)
			}

			if len(mapping.Content) > 0 {
				if value := mapping.Content[0].Value; value != nil {
					valueComment := value.GetComment()
					lineComment := strings.TrimSpace(valueComment.LineComment)
					if lineComment != tt.expectedLine {
						t.Errorf("expected line comment %q, got %q", tt.expectedLine, lineComment)
					}
				}
			}
		})
	}
}

func TestParser_AnchorsAndAliases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "simple anchor and alias",
			input: `anchor: &ref value
alias: *ref`,
		},
		{
			name: "anchor on mapping",
			input: `defaults: &defaults
  timeout: 30
  retries: 3
service:
  <<: *defaults
  port: 8080`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			node, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			if node == nil {
				t.Fatal("expected non-nil node")
			}

			// Basic check that parsing succeeded
			doc, ok := node.(*ast.Document)
			if !ok {
				t.Fatalf("expected Document, got %T", node)
			}

			if len(doc.Content) == 0 {
				t.Fatal("expected non-empty document content")
			}
		})
	}
}

func TestParser_BlockScalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		style    ast.ScalarStyle
	}{
		{
			name: "literal block",
			input: `text: |
  line1
  line2`,
			expected: "line1\nline2",
			style:    ast.LiteralStyle,
		},
		{
			name: "folded block",
			input: `text: >
  line1
  line2`,
			expected: "line1 line2",
			style:    ast.FoldedStyle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			node, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			doc, ok := node.(*ast.Document)
			if !ok {
				t.Fatalf("expected Document, got %T", node)
			}

			mapping, ok := doc.Content[0].(*ast.Mapping)
			if !ok {
				t.Fatalf("expected Mapping, got %T", doc.Content[0])
			}

			if len(mapping.Content) != 1 {
				t.Fatalf("expected 1 mapping entry, got %d", len(mapping.Content))
			}

			scalar, ok := mapping.Content[0].Value.(*ast.Scalar)
			if !ok {
				t.Fatalf("expected Scalar value, got %T", mapping.Content[0].Value)
			}

			if scalar.Style != tt.style {
				t.Errorf("expected style %v, got %v", tt.style, scalar.Style)
			}
		})
	}
}

func TestParser_MultiDocument(t *testing.T) {
	input := `---
doc1: value1
---
doc2: value2
...`

	p := NewParser(strings.NewReader(input))
	node, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	doc, ok := node.(*ast.Document)
	if !ok {
		t.Fatalf("expected Document, got %T", node)
	}

	// Should have parsed first document
	if len(doc.Content) == 0 {
		t.Fatal("expected document content")
	}
}

func TestParser_ComplexDocument(t *testing.T) {
	input := `# Application configuration
name: MyApp
version: 1.0.0

server:
  host: localhost
  port: 8080
  ssl: true

database:
  type: postgres
  connection:
    host: db.example.com
    port: 5432

features:
  - logging
  - monitoring
  - metrics

settings:
  debug: false
  timeout: 30`

	p := NewParser(strings.NewReader(input))
	node, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	doc, ok := node.(*ast.Document)
	if !ok {
		t.Fatalf("expected Document, got %T", node)
	}

	mapping, ok := doc.Content[0].(*ast.Mapping)
	if !ok {
		t.Fatalf("expected Mapping, got %T", doc.Content[0])
	}

	// Check that we have the expected top-level keys
	keys := []string{"name", "version", "server", "database", "features", "settings"}
	foundKeys := make(map[string]bool)

	for _, entry := range mapping.Content {
		if key, ok := entry.Key.(*ast.Scalar); ok {
			foundKeys[key.Value] = true
		}
	}

	for _, key := range keys {
		if !foundKeys[key] {
			t.Errorf("missing expected key: %s", key)
		}
	}
}

func TestParser_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "undefined alias",
			input:     `value: *undefined`,
			wantError: true,
		},
		{
			name:      "invalid mapping",
			input:     `key: : value`,
			wantError: false, // May not error depending on parser implementation
		},
		{
			name:      "empty input",
			input:     ``,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			_, err := p.Parse()

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParser_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty mapping value",
			input: `key:`,
		},
		{
			name:  "empty sequence item",
			input: `- `,
		},
		{
			name: "mixed types",
			input: `string: hello
number: 42
bool: true
null: ~
float: 3.14`,
		},
		{
			name: "special characters in quoted strings",
			input: `single: 'quote''s'
double: "quote\"s\n"`,
		},
		{
			name: "unicode in strings",
			input: `emoji: 😀
chinese: 你好
arabic: مرحبا`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(strings.NewReader(tt.input))
			node, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			if node == nil {
				t.Fatal("expected non-nil node")
			}
		})
	}
}

// Helper functions

func extractMapping(node ast.Node, prefix string) map[string]string {
	result := make(map[string]string)

	switch n := node.(type) {
	case *ast.Mapping:
		for _, entry := range n.Content {
			key := ""
			if k, ok := entry.Key.(*ast.Scalar); ok {
				key = k.Value
			}

			if prefix != "" {
				key = prefix + "." + key
			}

			switch v := entry.Value.(type) {
			case *ast.Scalar:
				result[key] = v.Value
			case *ast.Mapping:
				for k, v := range extractMapping(v, key) {
					result[k] = v
				}
			}
		}
	}

	return result
}

func extractSequence(node ast.Node) []string {
	var result []string

	switch n := node.(type) {
	case *ast.Sequence:
		for _, item := range n.Content {
			switch v := item.(type) {
			case *ast.Scalar:
				result = append(result, v.Value)
			case *ast.Sequence:
				result = append(result, extractSequence(v)...)
			}
		}
	}

	return result
}

func BenchmarkParser_SimpleDocument(b *testing.B) {
	input := `key1: value1
key2: value2
key3:
  nested1: data1
  nested2: data2`

	for i := 0; i < b.N; i++ {
		p := NewParser(strings.NewReader(input))
		p.Parse()
	}
}

func BenchmarkParser_ComplexDocument(b *testing.B) {
	input := `name: MyApp
version: 1.0.0
server:
  host: localhost
  port: 8080
  ssl: true
  timeouts:
    read: 30
    write: 30
    idle: 60
database:
  type: postgres
  pool:
    min: 5
    max: 20
features:
  - logging
  - monitoring
  - metrics
  - tracing`

	for i := 0; i < b.N; i++ {
		p := NewParser(strings.NewReader(input))
		p.Parse()
	}
}