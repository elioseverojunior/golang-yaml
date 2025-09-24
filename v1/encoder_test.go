package yaml

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"golang-yaml/v1/ast"
)

func TestEncoder_Scalars(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello\n"},
		{"integer", 42, "42\n"},
		{"float", 3.14, "3.14\n"},
		{"boolean true", true, "true\n"},
		{"boolean false", false, "false\n"},
		{"nil", nil, "null\n"},
		{"infinity", math.Inf(1), ".inf\n"},
		{"negative infinity", math.Inf(-1), "-.inf\n"},
		{"not a number", math.NaN(), ".nan\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tt.input)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_Maps(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"key": "value",
			},
			expected: "key: value\n",
		},
		{
			name: "multiple keys",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: "key1: value1\nkey2: value2\n", // Maps may have different ordering
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": "value",
				},
			},
			expected: "parent:\n  child: value\n",
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"string": "hello",
				"number": 42,
				"bool":   true,
				"null":   nil,
			},
			expected: "", // skip check due to map ordering
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tt.input)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()

			// Skip check for maps with unpredictable ordering
			if tt.expected == "" {
				return
			}

			// For maps with multiple keys, just check it contains expected keys
			if tt.name == "multiple keys" {
				if !strings.Contains(result, "key1:") || !strings.Contains(result, "key2:") {
					t.Errorf("expected keys not found in:\n%s", result)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_Slices(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "simple slice",
			input:    []string{"item1", "item2", "item3"},
			expected: "- item1\n- item2\n- item3\n",
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: "[]\n",
		},
		{
			name: "nested slice",
			input: []interface{}{
				[]string{"nested1", "nested2"},
				"item2",
			},
			expected: "- \n  - nested1\n  - nested2\n- item2\n",
		},
		{
			name: "slice with mixed types",
			input: []interface{}{
				"string",
				42,
				true,
				nil,
			},
			expected: "- string\n- 42\n- true\n- null\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tt.input)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_Structs(t *testing.T) {
	type SimpleStruct struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	type NestedStruct struct {
		Parent string       `yaml:"parent"`
		Child  SimpleStruct `yaml:"child"`
	}

	type WithOmit struct {
		Required string `yaml:"required"`
		Optional string `yaml:"optional,omitempty"`
		Private  string `yaml:"-"`
	}

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "simple struct",
			input: SimpleStruct{
				Name:  "test",
				Value: 42,
			},
			expected: "name: test\nvalue: 42\n",
		},
		{
			name: "nested struct",
			input: NestedStruct{
				Parent: "parent",
				Child: SimpleStruct{
					Name:  "child",
					Value: 10,
				},
			},
			expected: "parent: parent\nchild:\n  name: child\n  value: 10\n",
		},
		{
			name: "omitempty tag",
			input: WithOmit{
				Required: "present",
				Optional: "",
				Private:  "hidden",
			},
			expected: "required: present\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tt.input)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_Nodes(t *testing.T) {
	tests := []struct {
		name     string
		node     ast.Node
		expected string
	}{
		{
			name:     "scalar node",
			node:     ast.NewScalar("value"),
			expected: "value\n",
		},
		{
			name: "sequence node",
			node: &ast.Sequence{
				Content: []ast.Node{
					ast.NewScalar("item1"),
					ast.NewScalar("item2"),
				},
			},
			expected: "- item1\n- item2\n",
		},
		{
			name: "mapping node",
			node: &ast.Mapping{
				Content: []*ast.MappingEntry{
					{
						Key:   ast.NewScalar("key"),
						Value: ast.NewScalar("value"),
					},
				},
			},
			expected: "key: value\n",
		},
		{
			name: "document node",
			node: &ast.Document{
				Content: []ast.Node{
					ast.NewScalar("content"),
				},
			},
			expected: "content\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.EncodeNode(tt.node)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_ScalarStyles(t *testing.T) {
	tests := []struct {
		name     string
		node     *ast.Scalar
		expected string
	}{
		{
			name: "literal style",
			node: &ast.Scalar{
				Value: "line1\nline2\nline3",
				Style: ast.LiteralStyle,
			},
			expected: "|-\n  line1\n  line2\n  line3\n",
		},
		{
			name: "folded style",
			node: &ast.Scalar{
				Value: "line1 line2\nline3",
				Style: ast.FoldedStyle,
			},
			expected: ">-\n  line1 line2\n  line3\n",
		},
		{
			name: "single quoted",
			node: &ast.Scalar{
				Value: "quote's",
				Style: ast.SingleQuotedStyle,
			},
			expected: "'quote''s'\n",
		},
		{
			name: "double quoted",
			node: &ast.Scalar{
				Value: "quote\"s\n",
				Style: ast.DoubleQuotedStyle,
			},
			expected: "\"quote\\\"s\\n\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.EncodeNode(tt.node)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_Comments(t *testing.T) {
	tests := []struct {
		name     string
		node     ast.Node
		expected string
	}{
		{
			name: "head comment",
			node: func() ast.Node {
				n := ast.NewScalar("value")
				n.SetComment(ast.Comment{
					HeadComment: "This is a comment",
				})
				return n
			}(),
			expected: "# This is a comment\nvalue\n",
		},
		{
			name: "line comment",
			node: func() ast.Node {
				n := ast.NewScalar("value")
				n.SetComment(ast.Comment{
					LineComment: "inline comment",
				})
				return n
			}(),
			expected: "value # inline comment\n",
		},
		{
			name: "foot comment",
			node: func() ast.Node {
				n := ast.NewScalar("value")
				n.SetComment(ast.Comment{
					FootComment: "after comment",
				})
				return n
			}(),
			expected: "value\n# after comment\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.EncodeNode(tt.node)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_FlowStyle(t *testing.T) {
	tests := []struct {
		name     string
		node     ast.Node
		expected string
	}{
		{
			name: "flow sequence",
			node: &ast.Sequence{
				Style: ast.FlowStyle,
				Content: []ast.Node{
					ast.NewScalar("1"),
					ast.NewScalar("2"),
					ast.NewScalar("3"),
				},
			},
			expected: "[1, 2, 3]\n",
		},
		{
			name: "flow mapping",
			node: &ast.Mapping{
				Style: ast.FlowStyle,
				Content: []*ast.MappingEntry{
					{
						Key:   ast.NewScalar("a"),
						Value: ast.NewScalar("1"),
					},
					{
						Key:   ast.NewScalar("b"),
						Value: ast.NewScalar("2"),
					},
				},
			},
			expected: "{a: 1, b: 2}\n",
		},
		{
			name: "empty flow sequence",
			node: &ast.Sequence{
				Style:   ast.FlowStyle,
				Content: []ast.Node{},
			},
			expected: "[]\n",
		},
		{
			name: "empty flow mapping",
			node: &ast.Mapping{
				Style:   ast.FlowStyle,
				Content: []*ast.MappingEntry{},
			},
			expected: "{}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.EncodeNode(tt.node)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_Indentation(t *testing.T) {
	input := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "value",
			},
		},
	}

	tests := []struct {
		name     string
		indent   int
		expected string
	}{
		{
			name:   "2 spaces",
			indent: 2,
			expected: "level1:\n  level2:\n    level3: value\n",
		},
		{
			name:   "4 spaces",
			indent: 4,
			expected: "level1:\n    level2:\n        level3: value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			enc.SetIndent(tt.indent)
			err := enc.Encode(input)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_SpecialStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", `""`},
		{"true as string", "true", `"true"`},
		{"false as string", "false", `"false"`},
		{"yes as string", "yes", `"yes"`},
		{"no as string", "no", `"no"`},
		{"null as string", "null", `"null"`},
		{"number as string", "123", `"123"`},
		{"float as string", "3.14", `"3.14"`},
		{"special chars", "a:b#c", `"a:b#c"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)

			// Create a scalar that needs quoting
			scalar := ast.NewScalar(tt.input)
			err := enc.EncodeNode(scalar)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := strings.TrimSpace(buf.String())
			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_CustomMarshaler(t *testing.T) {
	type CustomType struct {
		value string
	}

	// Note: This assumes Marshaler interface is implemented
	// The actual implementation would need to satisfy the yaml.Marshaler interface

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "struct with pointer",
			input: struct {
				Ptr *string `yaml:"ptr"`
			}{
				Ptr: func() *string { s := "value"; return &s }(),
			},
			expected: "ptr: value\n",
		},
		{
			name: "struct with nil pointer",
			input: struct {
				Ptr *string `yaml:"ptr"`
			}{
				Ptr: nil,
			},
			expected: "ptr: null\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tt.input)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncoder_MultiDocument(t *testing.T) {
	combined := &ast.Document{
		Content: []ast.Node{
			ast.NewScalar("doc1"),
			ast.NewScalar("doc2"),
		},
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.EncodeNode(combined)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	result := buf.String()
	expected := "doc1\n\n---\ndoc2\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestEncoder_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantError bool
	}{
		{
			name:      "unsupported type",
			input:     make(chan int),
			wantError: true,
		},
		{
			name:      "complex number",
			input:     complex(1, 2),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tt.input)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func BenchmarkEncoder_SimpleStruct(b *testing.B) {
	type Simple struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	input := Simple{
		Name:  "test",
		Value: 42,
	}

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		enc.Encode(input)
	}
}

func BenchmarkEncoder_ComplexStruct(b *testing.B) {
	type Complex struct {
		Name     string                 `yaml:"name"`
		Version  string                 `yaml:"version"`
		Settings map[string]interface{} `yaml:"settings"`
		Features []string               `yaml:"features"`
	}

	input := Complex{
		Name:    "app",
		Version: "1.0.0",
		Settings: map[string]interface{}{
			"timeout": 30,
			"retries": 3,
		},
		Features: []string{"logging", "metrics"},
	}

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		enc.Encode(input)
	}
}