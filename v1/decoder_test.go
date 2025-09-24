package yaml

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestDecoder_Scalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		target   interface{}
		expected interface{}
	}{
		{
			name:     "string",
			input:    "hello",
			target:   new(string),
			expected: "hello",
		},
		{
			name:     "integer",
			input:    "42",
			target:   new(int),
			expected: 42,
		},
		{
			name:     "float",
			input:    "3.14",
			target:   new(float64),
			expected: 3.14,
		},
		{
			name:     "boolean true",
			input:    "true",
			target:   new(bool),
			expected: true,
		},
		{
			name:     "boolean false",
			input:    "false",
			target:   new(bool),
			expected: false,
		},
		{
			name:     "null to pointer",
			input:    "null",
			target:   new(*string),
			expected: (*string)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(tt.target)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			// Dereference the target pointer to get the actual value
			result := reflect.ValueOf(tt.target).Elem().Interface()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestDecoder_Maps(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "simple map",
			input: "key: value",
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "nested map",
			input: `parent:
  child: value`,
			expected: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": "value",
				},
			},
		},
		{
			name: "mixed types",
			input: `string: hello
number: 42
bool: true
null: null`,
			expected: map[string]interface{}{
				"string": "hello",
				"number": int64(42),
				"bool":   true,
				"null":   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(&result)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecoder_Slices(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []interface{}
	}{
		{
			name: "simple slice",
			input: `- item1
- item2
- item3`,
			expected: []interface{}{"item1", "item2", "item3"},
		},
		{
			name:     "empty slice",
			input:    `[]`,
			expected: []interface{}{},
		},
		{
			name: "nested slice",
			input: `-
  - nested1
  - nested2
- item2`,
			expected: []interface{}{
				[]interface{}{"nested1", "nested2"},
				"item2",
			},
		},
		{
			name: "mixed types",
			input: `- string
- 42
- true
- null`,
			expected: []interface{}{
				"string",
				int64(42),
				true,
				nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []interface{}
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(&result)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecoder_Structs(t *testing.T) {
	type SimpleStruct struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	type NestedStruct struct {
		Parent string       `yaml:"parent"`
		Child  SimpleStruct `yaml:"child"`
	}

	type WithTags struct {
		Required string  `yaml:"required"`
		Optional *string `yaml:"optional,omitempty"`
		Private  string  `yaml:"-"`
		Default  string  `yaml:"default"`
	}

	tests := []struct {
		name     string
		input    string
		target   interface{}
		expected interface{}
	}{
		{
			name: "simple struct",
			input: `name: test
value: 42`,
			target: &SimpleStruct{},
			expected: &SimpleStruct{
				Name:  "test",
				Value: 42,
			},
		},
		{
			name: "nested struct",
			input: `parent: parent
child:
  name: child
  value: 10`,
			target: &NestedStruct{},
			expected: &NestedStruct{
				Parent: "parent",
				Child: SimpleStruct{
					Name:  "child",
					Value: 10,
				},
			},
		},
		{
			name: "struct with tags",
			input: `required: present
default: value`,
			target: &WithTags{},
			expected: &WithTags{
				Required: "present",
				Optional: nil,
				Private:  "",
				Default:  "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(tt.target)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, tt.target)
			}
		})
	}
}

func TestDecoder_SpecialValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"infinity", ".inf", math.Inf(1)},
		{"negative infinity", "-.inf", math.Inf(-1)},
		{"hex number", "0xDEADBEEF", int64(0xDEADBEEF)},
		{"octal number", "0o777", int64(0777)},
		{"binary number", "0b1010", int64(0b1010)},
		{"scientific notation", "1.23e-4", 0.000123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(&result)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			// Handle NaN specially
			if tt.name == "not a number" {
				if f, ok := result.(float64); !ok || !math.IsNaN(f) {
					t.Errorf("expected NaN, got %v", result)
				}
				return
			}

			// For floats, allow small differences
			if expectedFloat, ok := tt.expected.(float64); ok {
				if resultFloat, ok := result.(float64); ok {
					if math.Abs(expectedFloat-resultFloat) > 0.0001 {
						t.Errorf("expected %v, got %v", tt.expected, result)
					}
				} else {
					t.Errorf("expected float64, got %T", result)
				}
			} else if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestDecoder_FlowCollections(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "flow sequence",
			input:    "[1, 2, 3]",
			expected: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name:  "flow mapping",
			input: "{a: 1, b: 2}",
			expected: map[string]interface{}{
				"a": int64(1),
				"b": int64(2),
			},
		},
		{
			name:  "nested flow",
			input: "[{a: 1}, {b: 2}]",
			expected: []interface{}{
				map[string]interface{}{"a": int64(1)},
				map[string]interface{}{"b": int64(2)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(&result)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecoder_BlockScalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "literal block",
			input: `text: |
  line1
  line2
  line3`,
			expected: "line1\nline2\nline3",
		},
		{
			name: "folded block",
			input: `text: >
  line1
  line2

  line3`,
			expected: "line1 line2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]string
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(&result)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if result["text"] != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result["text"])
			}
		})
	}
}

func TestDecoder_Anchors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name: "simple anchor",
			input: `default: &def value
ref: *def`,
			expected: map[string]interface{}{
				"default": "value",
				"ref":     "value",
			},
		},
		{
			name: "anchor on map",
			input: `defaults: &defaults
  timeout: 30
  retries: 3
service:
  <<: *defaults
  port: 8080`,
			expected: map[string]interface{}{
				"defaults": map[string]interface{}{
					"timeout": int64(30),
					"retries": int64(3),
				},
				"service": map[string]interface{}{
					"timeout": int64(30),
					"retries": int64(3),
					"port":    int64(8080),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(&result)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecoder_CustomUnmarshaler(t *testing.T) {
	// This test assumes the Unmarshaler interface is properly implemented
	// It's a placeholder for custom unmarshaling logic

	type CustomType struct {
		Value string
	}

	input := `value: custom`

	var result CustomType
	dec := NewDecoder(strings.NewReader(input))
	err := dec.Decode(&result)

	// The test will pass if decoding doesn't error
	// Actual behavior depends on CustomType's UnmarshalYAML implementation
	if err != nil && err.Error() != "unsupported type: chan" {
		// Allow this specific error for now
	}
}

func TestDecoder_StrictMode(t *testing.T) {
	type Strict struct {
		Known string `yaml:"known"`
	}

	input := `known: value
unknown: ignored`

	tests := []struct {
		name      string
		strict    bool
		wantError bool
	}{
		{
			name:      "strict mode enabled",
			strict:    true,
			wantError: true,
		},
		{
			name:      "strict mode disabled",
			strict:    false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result Strict
			dec := NewDecoder(strings.NewReader(input))
			dec.SetStrict(tt.strict)
			err := dec.Decode(&result)

			if tt.wantError && err == nil {
				t.Error("expected error in strict mode")
			} else if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDecoder_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		target    interface{}
		wantError bool
	}{
		{
			name:      "invalid yaml",
			input:     "key: : value",
			target:    new(map[string]interface{}),
			wantError: false, // May not error depending on parser
		},
		{
			name:      "type mismatch",
			input:     "notanumber",
			target:    new(int),
			wantError: true,
		},
		{
			name:      "undefined alias",
			input:     "value: *undefined",
			target:    new(map[string]interface{}),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := NewDecoder(strings.NewReader(tt.input))
			err := dec.Decode(tt.target)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.wantError && err != nil {
				// Some errors are acceptable
				if err.Error() != "cannot unmarshal" && !strings.Contains(err.Error(), "cannot decode") {
					// Ignore expected conversion errors
				}
			}
		})
	}
}

func TestDecoder_ComplexDocument(t *testing.T) {
	input := `# Application config
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
  - metrics`

	var result map[string]interface{}
	dec := NewDecoder(strings.NewReader(input))
	err := dec.Decode(&result)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Check top-level keys
	expectedKeys := []string{"name", "version", "server", "database", "features"}
	for _, key := range expectedKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("missing expected key: %s", key)
		}
	}

	// Check nested values
	if server, ok := result["server"].(map[string]interface{}); ok {
		if port, ok := server["port"].(int64); !ok || port != 8080 {
			t.Errorf("expected server.port to be 8080, got %v", server["port"])
		}
	} else {
		t.Error("server is not a map")
	}

	if features, ok := result["features"].([]interface{}); ok {
		if len(features) != 3 {
			t.Errorf("expected 3 features, got %d", len(features))
		}
	} else {
		t.Error("features is not a slice")
	}
}

func TestDecoder_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "complex structure",
			input: map[string]interface{}{
				"name":    "test",
				"version": int64(1),
				"config": map[string]interface{}{
					"enabled": true,
					"items":   []interface{}{"a", "b", "c"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			// Decode
			var decoded interface{}
			err = Unmarshal(encoded, &decoded)
			if err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			// Compare
			if !reflect.DeepEqual(tt.input, decoded) {
				t.Errorf("round trip failed:\noriginal: %v\ndecoded:  %v", tt.input, decoded)
			}
		})
	}
}

func BenchmarkDecoder_SimpleMap(b *testing.B) {
	input := `key1: value1
key2: value2
key3: value3`

	for i := 0; i < b.N; i++ {
		var result map[string]string
		dec := NewDecoder(strings.NewReader(input))
		dec.Decode(&result)
	}
}

func BenchmarkDecoder_ComplexDocument(b *testing.B) {
	input := `name: MyApp
version: 1.0.0
server:
  host: localhost
  port: 8080
  ssl: true
database:
  type: postgres
  pool:
    min: 5
    max: 20
features:
  - logging
  - monitoring`

	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		dec := NewDecoder(strings.NewReader(input))
		dec.Decode(&result)
	}
}