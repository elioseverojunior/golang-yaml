package lexer

import "fmt"

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenNewLine
	TokenIndent
	TokenDedent
	TokenDocumentStart
	TokenDocumentEnd
	TokenKey
	TokenValue
	TokenString
	TokenNumber
	TokenBoolean
	TokenNull
	TokenSequenceStart
	TokenSequenceItem
	TokenMappingStart
	TokenAnchor
	TokenAlias
	TokenTag
	TokenComment
	TokenLiteralBlock
	TokenFoldedBlock
	TokenFlowSequenceStart
	TokenFlowSequenceEnd
	TokenFlowMappingStart
	TokenFlowMappingEnd
	TokenFlowEntry
	TokenError
)

type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
	Offset int
}

func (t TokenType) String() string {
	names := map[TokenType]string{
		TokenEOF:               "EOF",
		TokenNewLine:           "NewLine",
		TokenIndent:            "Indent",
		TokenDedent:            "Dedent",
		TokenDocumentStart:     "DocumentStart",
		TokenDocumentEnd:       "DocumentEnd",
		TokenKey:               "Key",
		TokenValue:             "Value",
		TokenString:            "String",
		TokenNumber:            "Number",
		TokenBoolean:           "Boolean",
		TokenNull:              "Null",
		TokenSequenceStart:     "SequenceStart",
		TokenSequenceItem:      "SequenceItem",
		TokenMappingStart:      "MappingStart",
		TokenAnchor:            "Anchor",
		TokenAlias:             "Alias",
		TokenTag:               "Tag",
		TokenComment:           "Comment",
		TokenLiteralBlock:      "LiteralBlock",
		TokenFoldedBlock:       "FoldedBlock",
		TokenFlowSequenceStart: "FlowSequenceStart",
		TokenFlowSequenceEnd:   "FlowSequenceEnd",
		TokenFlowMappingStart:  "FlowMappingStart",
		TokenFlowMappingEnd:    "FlowMappingEnd",
		TokenFlowEntry:         "FlowEntry",
		TokenError:             "Error",
	}

	if name, ok := names[t]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", t)
}

func (t Token) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%q) at %d:%d", t.Type, t.Value, t.Line, t.Column)
	}
	return fmt.Sprintf("%s at %d:%d", t.Type, t.Line, t.Column)
}
