package yaml

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

func TestBasicUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want interface{}
	}{
		{
			name: "simple string",
			yaml: "hello world",
			want: "hello world",
		},
		{
			name: "simple number",
			yaml: "42",
			want: int64(42),
		},
		{
			name: "simple bool",
			yaml: "true",
			want: true,
		},
		{
			name: "simple null",
			yaml: "null",
			want: nil,
		},
		{
			name: "simple map",
			yaml: "key: value",
			want: map[string]interface{}{"key": "value"},
		},
		{
			name: "simple array",
			yaml: "- one\n- two\n- three",
			want: []interface{}{"one", "two", "three"},
		},
		{
			name: "nested map",
			yaml: "parent:\n  child: value",
			want: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got interface{}
			err := Unmarshal([]byte(tt.yaml), &got)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	type TestStruct struct {
		Name    string   `yaml:"name"`
		Age     int      `yaml:"age"`
		Tags    []string `yaml:"tags"`
		Enabled bool     `yaml:"enabled"`
	}

	original := TestStruct{
		Name:    "Test",
		Age:     30,
		Tags:    []string{"tag1", "tag2"},
		Enabled: true,
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded TestStruct
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("Round-trip failed: got = %v, want %v", decoded, original)
	}
}

func TestMerge(t *testing.T) {
	base := `
name: base
version: 1.0.0
config:
  timeout: 30
  retries: 3
`

	override := `
version: 2.0.0
config:
  timeout: 60
  debug: true
`

	expected := map[string]interface{}{
		"name":    "base",
		"version": "2.0.0",
		"config": map[string]interface{}{
			"timeout": int64(60),
			"retries": int64(3),
			"debug":   true,
		},
	}

	merged, err := Merge([]byte(base), []byte(override), MergeOptions{
		Mode:               MergeDeep,
		ArrayMergeStrategy: ArrayReplace,
	})
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	var result interface{}
	err = Unmarshal(merged, &result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Merge() got = %v, want %v", result, expected)
	}
}

func TestYAML12Features(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want interface{}
	}{
		{
			name: "octal number",
			yaml: "0o10",
			want: int64(8),
		},
		{
			name: "hex number",
			yaml: "0xFF",
			want: int64(255),
		},
		{
			name: "binary number",
			yaml: "0b1010",
			want: int64(10),
		},
		{
			name: "infinity",
			yaml: ".inf",
			want: math.Inf(1),
		},
		{
			name: "negative infinity",
			yaml: "-.inf",
			want: math.Inf(-1),
		},
		{
			name: "boolean yes",
			yaml: "yes",
			want: true,
		},
		{
			name: "boolean no",
			yaml: "no",
			want: false,
		},
		{
			name: "null tilde",
			yaml: "~",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got interface{}
			err := Unmarshal([]byte(tt.yaml), &got)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			switch v := tt.want.(type) {
			case float64:
				gotFloat, ok := got.(float64)
				if !ok {
					t.Errorf("Expected float64, got %T", got)
					return
				}
				if math.IsNaN(v) {
					if !math.IsNaN(gotFloat) {
						t.Errorf("Expected NaN, got %v", gotFloat)
					}
				} else if math.IsInf(v, 1) {
					if !math.IsInf(gotFloat, 1) {
						t.Errorf("Expected +Inf, got %v", gotFloat)
					}
				} else if math.IsInf(v, -1) {
					if !math.IsInf(gotFloat, -1) {
						t.Errorf("Expected -Inf, got %v", gotFloat)
					}
				} else if gotFloat != v {
					t.Errorf("Got = %v, want %v", gotFloat, v)
				}
			default:
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestFlowStyle(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want interface{}
	}{
		{
			name: "flow sequence",
			yaml: "[1, 2, 3]",
			want: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name: "flow mapping",
			yaml: "{key1: value1, key2: value2}",
			want: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "nested flow",
			yaml: "{array: [1, 2], map: {a: b}}",
			want: map[string]interface{}{
				"array": []interface{}{int64(1), int64(2)},
				"map":   map[string]interface{}{"a": "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got interface{}
			err := Unmarshal([]byte(tt.yaml), &got)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockScalars(t *testing.T) {
	literalYAML := `literal: |
  Line 1
  Line 2
    Indented line`

	foldedYAML := `folded: >
  This is
  a folded
  scalar.`

	t.Run("literal block", func(t *testing.T) {
		var result map[string]interface{}
		err := Unmarshal([]byte(literalYAML), &result)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		expected := "Line 1\nLine 2\n  Indented line\n"
		if result["literal"] != expected {
			t.Errorf("Got = %q, want %q", result["literal"], expected)
		}
	})

	t.Run("folded block", func(t *testing.T) {
		var result map[string]interface{}
		err := Unmarshal([]byte(foldedYAML), &result)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		expected := "This is a folded scalar.\n"
		if result["folded"] != expected {
			t.Errorf("Got = %q, want %q", result["folded"], expected)
		}
	})
}

func TestDocumentMarkers(t *testing.T) {
	yaml := `---
doc: 1
...
---
doc: 2`

	reader := bytes.NewReader([]byte(yaml))
	decoder := NewDecoder(reader)

	var doc1 interface{}
	err := decoder.Decode(&doc1)
	if err != nil {
		t.Fatalf("Failed to decode first document: %v", err)
	}

	expected1 := map[string]interface{}{"doc": int64(1)}
	if !reflect.DeepEqual(doc1, expected1) {
		t.Errorf("First document: got = %v, want %v", doc1, expected1)
	}

}
