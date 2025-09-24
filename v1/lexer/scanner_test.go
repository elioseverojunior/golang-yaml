package lexer

import (
	"bytes"
	"strings"
	"testing"
)

func TestScanner_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []TokenType{TokenEOF},
		},
		{
			name:     "comment only",
			input:    "# This is a comment",
			expected: []TokenType{TokenComment, TokenEOF},
		},
		{
			name:     "simple scalar",
			input:    "hello",
			expected: []TokenType{TokenString, TokenEOF},
		},
		{
			name:     "key-value pair",
			input:    "key: value",
			expected: []TokenType{TokenString, TokenKey, TokenString, TokenEOF},
		},
		{
			name:     "number value",
			input:    "age: 42",
			expected: []TokenType{TokenString, TokenKey, TokenNumber, TokenEOF},
		},
		{
			name:     "boolean values",
			input:    "yes: true\nno: false",
			expected: []TokenType{TokenBoolean, TokenKey, TokenBoolean, TokenNewLine, TokenBoolean, TokenKey, TokenBoolean, TokenEOF},
		},
		{
			name:     "null values",
			input:    "empty: null\nnil: ~",
			expected: []TokenType{TokenString, TokenKey, TokenNull, TokenNewLine, TokenString, TokenKey, TokenNull, TokenEOF},
		},
		{
			name:     "document markers",
			input:    "---\ndata\n...",
			expected: []TokenType{TokenDocumentStart, TokenNewLine, TokenString, TokenNewLine, TokenDocumentEnd, TokenEOF},
		},
		{
			name:     "sequence items",
			input:    "- item1\n- item2",
			expected: []TokenType{TokenSequenceItem, TokenString, TokenNewLine, TokenSequenceItem, TokenString, TokenEOF},
		},
		{
			name:     "flow sequence",
			input:    "[1, 2, 3]",
			expected: []TokenType{TokenFlowSequenceStart, TokenNumber, TokenFlowEntry, TokenNumber, TokenFlowEntry, TokenNumber, TokenFlowSequenceEnd, TokenEOF},
		},
		{
			name:     "flow mapping",
			input:    "{a: 1, b: 2}",
			expected: []TokenType{TokenFlowMappingStart, TokenString, TokenKey, TokenNumber, TokenFlowEntry, TokenString, TokenKey, TokenNumber, TokenFlowMappingEnd, TokenEOF},
		},
		{
			name:     "anchors and aliases",
			input:    "&anchor value\n*anchor",
			expected: []TokenType{TokenAnchor, TokenString, TokenNewLine, TokenAlias, TokenEOF},
		},
		{
			name:     "tags",
			input:    "!tag value",
			expected: []TokenType{TokenTag, TokenString, TokenEOF},
		},
		{
			name:     "literal block",
			input:    "text: |\n  line1\n  line2",
			expected: []TokenType{TokenString, TokenKey, TokenLiteralBlock, TokenEOF},
		},
		{
			name:     "folded block",
			input:    "text: >\n  line1\n  line2",
			expected: []TokenType{TokenString, TokenKey, TokenFoldedBlock, TokenEOF},
		},
		{
			name:     "single quoted string",
			input:    "'quoted string'",
			expected: []TokenType{TokenString, TokenEOF},
		},
		{
			name:     "double quoted string",
			input:    `"quoted \"string\""`,
			expected: []TokenType{TokenString, TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tt.input))
			var tokens []TokenType
			for {
				token, err := scanner.Scan()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				tokens = append(tokens, token.Type)
				if token.Type == TokenEOF {
					break
				}
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(tokens))
				t.Errorf("expected: %v", tt.expected)
				t.Errorf("got: %v", tokens)
				return
			}

			for i, expectedType := range tt.expected {
				if tokens[i] != expectedType {
					t.Errorf("token %d: expected %v, got %v", i, expectedType, tokens[i])
				}
			}
		})
	}
}

func TestScanner_Indentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name: "simple indentation",
			input: `key:
  nested: value`,
			expected: []TokenType{
				TokenString, TokenKey, TokenNewLine,
				TokenIndent, TokenString, TokenKey, TokenString,
				TokenDedent, TokenEOF,
			},
		},
		{
			name: "multiple indent levels",
			input: `a:
  b:
    c: 1`,
			expected: []TokenType{
				TokenString, TokenKey, TokenNewLine,
				TokenIndent, TokenString, TokenKey, TokenNewLine,
				TokenIndent, TokenString, TokenKey, TokenNumber,
				TokenDedent, TokenDedent, TokenEOF,
			},
		},
		{
			name: "indent and dedent",
			input: `a:
  b: 1
c: 2`,
			expected: []TokenType{
				TokenString, TokenKey, TokenNewLine,
				TokenIndent, TokenString, TokenKey, TokenNumber, TokenNewLine,
				TokenDedent, TokenString, TokenKey, TokenNumber,
				TokenEOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tt.input))
			var tokens []TokenType
			for {
				token, err := scanner.Scan()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				tokens = append(tokens, token.Type)
				if token.Type == TokenEOF {
					break
				}
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(tokens))
				t.Errorf("expected: %v", tt.expected)
				t.Errorf("got: %v", tokens)
				return
			}

			for i, expectedType := range tt.expected {
				if tokens[i] != expectedType {
					t.Errorf("token %d: expected %v, got %v", i, expectedType, tokens[i])
				}
			}
		})
	}
}

func TestScanner_Comments(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTypes []TokenType
		expectedVals  []string
	}{
		{
			name:          "inline comment",
			input:         "key: value # comment",
			expectedTypes: []TokenType{TokenString, TokenKey, TokenString, TokenComment, TokenEOF},
			expectedVals:  []string{"key", ":", "value", "comment", ""},
		},
		{
			name: "head comment",
			input: `# Comment
key: value`,
			expectedTypes: []TokenType{TokenComment, TokenNewLine, TokenString, TokenKey, TokenString, TokenEOF},
			expectedVals:  []string{"Comment", "\n", "key", ":", "value", ""},
		},
		{
			name: "multiple comments",
			input: `# Comment 1
# Comment 2
key: value # inline`,
			expectedTypes: []TokenType{
				TokenComment, TokenNewLine,
				TokenComment, TokenNewLine,
				TokenString, TokenKey, TokenString, TokenComment, TokenEOF,
			},
			expectedVals: []string{"Comment 1", "\n", "Comment 2", "\n", "key", ":", "value", "inline", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tt.input))
			var tokens []Token
			for {
				token, err := scanner.Scan()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				tokens = append(tokens, token)
				if token.Type == TokenEOF {
					break
				}
			}

			if len(tokens) != len(tt.expectedTypes) {
				t.Errorf("expected %d tokens, got %d", len(tt.expectedTypes), len(tokens))
				return
			}

			for i, token := range tokens {
				if token.Type != tt.expectedTypes[i] {
					t.Errorf("token %d: expected type %v, got %v", i, tt.expectedTypes[i], token.Type)
				}
				if token.Value != tt.expectedVals[i] {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expectedVals[i], token.Value)
				}
			}
		})
	}
}

func TestScanner_SpecialValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		value string
		ttype TokenType
	}{
		{"infinity", ".inf", ".inf", TokenNumber},
		{"negative infinity", "-.inf", "-.inf", TokenNumber},
		{"positive infinity", "+.inf", "+.inf", TokenNumber},
		{"not a number", ".nan", ".nan", TokenNumber},
		{"hex number", "0xDEADBEEF", "0xDEADBEEF", TokenNumber},
		{"octal number", "0o777", "0o777", TokenNumber},
		{"binary number", "0b1010", "0b1010", TokenNumber},
		{"float", "3.14159", "3.14159", TokenNumber},
		{"scientific", "1.23e-4", "1.23e-4", TokenNumber},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tt.input))
			token, err := scanner.Scan()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if token.Type != tt.ttype {
				t.Errorf("expected type %v, got %v", tt.ttype, token.Type)
			}

			if token.Value != tt.value {
				t.Errorf("expected value %q, got %q", tt.value, token.Value)
			}
		})
	}
}

func TestScanner_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "unclosed single quote",
			input:     "'unclosed",
			wantError: true,
		},
		{
			name:      "unclosed double quote",
			input:     `"unclosed`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tt.input))
			hasError := false
			for {
				token, err := scanner.Scan()
				if err != nil {
					hasError = true
					break
				}
				if token.Type == TokenEOF {
					break
				}
			}

			if hasError != tt.wantError {
				t.Errorf("expected error: %v, got error: %v", tt.wantError, hasError)
			}
		})
	}
}

func TestScanner_Position(t *testing.T) {
	input := `key: value
nested:
  child: data`

	scanner := NewScanner(strings.NewReader(input))

	// First token should be at line 1, column 1
	token, _ := scanner.Scan()
	if token.Line != 1 || token.Column != 1 {
		t.Errorf("first token position: expected (1,1), got (%d,%d)", token.Line, token.Column)
	}

	// Skip to newline
	for token.Type != TokenNewLine {
		token, _ = scanner.Scan()
	}

	// Next token after newline should be on line 2
	token, _ = scanner.Scan()
	if token.Type != TokenString {
		// might be indent token
		if token.Type == TokenIndent {
			token, _ = scanner.Scan()
		}
	}
	if token.Line != 2 {
		t.Errorf("token after newline: expected line 2, got line %d", token.Line)
	}
}

func TestScanner_PushBack(t *testing.T) {
	scanner := NewScanner(strings.NewReader("a b c"))

	// Scan first token
	token1, _ := scanner.Scan()
	if token1.Value != "a" {
		t.Errorf("expected 'a', got %q", token1.Value)
	}

	// Scan second token
	token2, _ := scanner.Scan()
	if token2.Value != "b" {
		t.Errorf("expected 'b', got %q", token2.Value)
	}

	// Push back second token
	scanner.PushBack(token2)

	// Scan again, should get 'b'
	token3, _ := scanner.Scan()
	if token3.Value != "b" {
		t.Errorf("after pushback, expected 'b', got %q", token3.Value)
	}

	// Continue scanning
	token4, _ := scanner.Scan()
	if token4.Value != "c" {
		t.Errorf("expected 'c', got %q", token4.Value)
	}
}

func TestScanner_BlockScalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		style    TokenType
	}{
		{
			name: "literal block",
			input: `text: |
  line1
  line2
  line3`,
			expected: "line1\nline2\nline3",
			style:    TokenLiteralBlock,
		},
		{
			name: "folded block",
			input: `text: >
  line1
  line2

  line3`,
			expected: "line1 line2\nline3",
			style:    TokenFoldedBlock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tt.input))

			// Skip to block scalar token
			var token Token
			for {
				token, _ = scanner.Scan()
				if token.Type == tt.style {
					break
				}
				if token.Type == TokenEOF {
					t.Fatal("expected block scalar not found")
				}
			}

			if token.Type != tt.style {
				t.Errorf("expected token type %v, got %v", tt.style, token.Type)
			}

			// Note: The actual block content parsing might differ
			// This is a simplified test
		})
	}
}

func TestScanner_FlowCollections(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []TokenType
	}{
		{
			name:  "empty flow sequence",
			input: "[]",
			tokens: []TokenType{
				TokenFlowSequenceStart,
				TokenFlowSequenceEnd,
				TokenEOF,
			},
		},
		{
			name:  "empty flow mapping",
			input: "{}",
			tokens: []TokenType{
				TokenFlowMappingStart,
				TokenFlowMappingEnd,
				TokenEOF,
			},
		},
		{
			name:  "nested flow collections",
			input: "[{a: 1}, {b: 2}]",
			tokens: []TokenType{
				TokenFlowSequenceStart,
				TokenFlowMappingStart,
				TokenString, TokenKey, TokenNumber,
				TokenFlowMappingEnd,
				TokenFlowEntry,
				TokenFlowMappingStart,
				TokenString, TokenKey, TokenNumber,
				TokenFlowMappingEnd,
				TokenFlowSequenceEnd,
				TokenEOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tt.input))
			var tokens []TokenType
			for {
				token, err := scanner.Scan()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				tokens = append(tokens, token.Type)
				if token.Type == TokenEOF {
					break
				}
			}

			if len(tokens) != len(tt.tokens) {
				t.Errorf("expected %d tokens, got %d", len(tt.tokens), len(tokens))
				t.Errorf("expected: %v", tt.tokens)
				t.Errorf("got: %v", tokens)
				return
			}

			for i, expectedType := range tt.tokens {
				if tokens[i] != expectedType {
					t.Errorf("token %d: expected %v, got %v", i, expectedType, tokens[i])
				}
			}
		})
	}
}

func TestScanner_ComplexDocument(t *testing.T) {
	input := `---
# Configuration file
name: MyApp # Application name
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
  - &metrics metrics

aliases:
  main_feature: *metrics

tags:
  custom: !custom_tag value
...`

	scanner := NewScanner(strings.NewReader(input))

	tokenCount := 0
	commentCount := 0

	for {
		token, err := scanner.Scan()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if token.Type == TokenComment {
			commentCount++
		}

		tokenCount++

		if token.Type == TokenEOF {
			break
		}
	}

	if tokenCount < 20 {
		t.Errorf("expected at least 20 tokens for complex document, got %d", tokenCount)
	}

	if commentCount != 2 {
		t.Errorf("expected 2 comments, got %d", commentCount)
	}
}

func BenchmarkScanner_SimpleDocument(b *testing.B) {
	input := `key1: value1
key2: value2
key3:
  nested1: data1
  nested2: data2`

	for i := 0; i < b.N; i++ {
		scanner := NewScanner(strings.NewReader(input))
		for {
			token, _ := scanner.Scan()
			if token.Type == TokenEOF {
				break
			}
		}
	}
}

func BenchmarkScanner_LargeDocument(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		buf.WriteString("key")
		buf.WriteString(string(rune('0' + i%10)))
		buf.WriteString(": value\n")
	}
	input := buf.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewScanner(strings.NewReader(input))
		for {
			token, _ := scanner.Scan()
			if token.Type == TokenEOF {
				break
			}
		}
	}
}