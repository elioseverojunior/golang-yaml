package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang-yaml/v1/ast"
	"golang-yaml/v1/lexer"
)

var debug = false

type Parser struct {
	scanner      *lexer.Scanner
	currentToken lexer.Token
	anchors      map[string]ast.Node
	comments     []lexer.Token
	indentLevel  int // Track current indentation level
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		scanner:  lexer.NewScanner(r),
		anchors:  make(map[string]ast.Node),
		comments: make([]lexer.Token, 0),
	}
}

func (p *Parser) Parse() (ast.Node, error) {
	token, err := p.scanner.Scan()
	if err != nil {
		return nil, err
	}
	p.currentToken = token
	if debug {
		fmt.Printf("Parse: initial token = %v\n", p.currentToken)
	}

	doc := ast.NewDocument()

	// Check if the document starts with a mapping at column 1
	if p.isMapping() && p.currentToken.Column == 1 {
		// Parse as a single root mapping
		mapping, err := p.parseMapping()
		if err != nil {
			return nil, err
		}
		doc.Content = append(doc.Content, mapping)
	} else {
		// Parse multiple values
		for p.currentToken.Type != lexer.TokenEOF {
			if debug {
				fmt.Printf("Parse loop: currentToken = %v\n", p.currentToken)
			}
			if p.currentToken.Type == lexer.TokenDocumentStart {
				p.advance()
			}

			if p.currentToken.Type == lexer.TokenDocumentEnd {
				p.advance()
				continue
			}

			node, err := p.parseValue()
			if err != nil {
				return nil, err
			}

			if node != nil {
				doc.Content = append(doc.Content, node)
			}

			p.skipNewlines()
		}
	}

	return doc, nil
}

func (p *Parser) parseValue() (ast.Node, error) {
	p.skipNewlines()
	p.collectComments()

	if debug {
		fmt.Printf("parseValue: currentToken = %v\n", p.currentToken)
	}

	switch p.currentToken.Type {
	case lexer.TokenEOF:
		return nil, nil

	case lexer.TokenDocumentEnd:
		return nil, nil

	case lexer.TokenNull:
		node := ast.NewScalar("")
		node.SetTag("!!null")
		p.attachComments(node)
		p.advance()
		return node, nil

	case lexer.TokenBoolean:
		// Check if this is actually the start of a mapping
		if p.isMapping() {
			return p.parseMapping()
		}
		node := ast.NewScalar(p.currentToken.Value)
		node.SetTag("!!bool")
		p.attachComments(node)
		p.advance()
		return node, nil

	case lexer.TokenNumber:
		// Check if this is actually the start of a mapping
		if p.isMapping() {
			return p.parseMapping()
		}
		node := p.parseNumber()
		p.attachComments(node)
		p.advance()
		return node, nil

	case lexer.TokenString:
		if debug {
			fmt.Printf("parseValue: TokenString case\n")
		}
		// Check if this is actually the start of a mapping
		if p.isMapping() {
			if debug {
				fmt.Printf("parseValue: TokenString but isMapping true, calling parseMapping\n")
			}
			return p.parseMapping()
		}
		node := ast.NewScalar(p.currentToken.Value)
		node.SetTag("!!str")
		p.attachComments(node)
		p.advance()
		return node, nil

	case lexer.TokenLiteralBlock:
		node := ast.NewScalar(p.currentToken.Value)
		node.Style = ast.LiteralStyle
		node.SetTag("!!str")
		p.attachComments(node)
		p.advance()
		return node, nil

	case lexer.TokenFoldedBlock:
		node := ast.NewScalar(p.currentToken.Value)
		node.Style = ast.FoldedStyle
		node.SetTag("!!str")
		p.attachComments(node)
		p.advance()
		return node, nil

	case lexer.TokenSequenceItem:
		return p.parseSequence()

	case lexer.TokenFlowSequenceStart:
		return p.parseFlowSequence()

	case lexer.TokenFlowMappingStart:
		return p.parseFlowMapping()

	case lexer.TokenAnchor:
		anchorName := p.currentToken.Value
		p.advance()
		node, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		p.anchors[anchorName] = node
		return node, nil

	case lexer.TokenAlias:
		aliasName := p.currentToken.Value
		p.advance()
		if node, ok := p.anchors[aliasName]; ok {
			return node.Clone(), nil
		}
		return nil, fmt.Errorf("undefined alias: %s", aliasName)

	case lexer.TokenTag:
		tag := p.currentToken.Value
		p.advance()
		node, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if node != nil {
			node.SetTag(tag)
		}
		return node, nil


	default:
		if debug {
			fmt.Printf("parseValue: default case\n")
		}
		if p.isMapping() {
			if debug {
				fmt.Printf("parseValue: isMapping returned true, calling parseMapping\n")
			}
			return p.parseMapping()
		}

		if p.currentToken.Type == lexer.TokenString {
			node := ast.NewScalar(p.currentToken.Value)
			node.SetTag("!!str")
			p.attachComments(node)
			p.advance()
			return node, nil
		}

		return nil, fmt.Errorf("unexpected token: %s", p.currentToken.Type)
	}
}

func (p *Parser) parseSequence() (ast.Node, error) {
	sequence := ast.NewSequence()
	p.attachComments(sequence)

	for p.currentToken.Type == lexer.TokenSequenceItem {
		p.advance()
		p.skipNewlines()
		p.collectComments()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if value != nil {
			sequence.Content = append(sequence.Content, value)
		}

		p.skipNewlines()
	}

	return sequence, nil
}

func (p *Parser) parseFlowSequence() (ast.Node, error) {
	sequence := ast.NewSequence()
	sequence.Style = ast.FlowStyle
	p.attachComments(sequence)
	p.advance()

	for p.currentToken.Type != lexer.TokenFlowSequenceEnd {
		p.skipNewlines()
		p.collectComments()

		if p.currentToken.Type == lexer.TokenFlowSequenceEnd {
			break
		}

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if value != nil {
			sequence.Content = append(sequence.Content, value)
		}

		p.skipNewlines()

		if p.currentToken.Type == lexer.TokenFlowEntry {
			p.advance()
			p.skipNewlines()
		}
	}

	if p.currentToken.Type == lexer.TokenFlowSequenceEnd {
		p.advance()
	}

	return sequence, nil
}

func (p *Parser) parseMapping() (ast.Node, error) {
	mapping := ast.NewMapping()
	p.attachComments(mapping)
	if debug {
		fmt.Printf("parseMapping: starting, currentToken = %v\n", p.currentToken)
	}

	// Remember the indentation level when we started this mapping
	// For nested mappings, we need to track the actual indentation of the first key
	var startColumn int
	var isRootMapping bool
	firstKey := true

	for p.currentToken.Type != lexer.TokenEOF && p.currentToken.Type != lexer.TokenDocumentEnd {
		p.skipNewlines()
		p.collectComments()

		if p.currentToken.Type == lexer.TokenEOF || p.currentToken.Type == lexer.TokenDocumentEnd {
			break
		}

		if p.currentToken.Type == lexer.TokenSequenceItem {
			break
		}

		// Set the indentation level based on the first key
		if firstKey {
			startColumn = p.currentToken.Column
			isRootMapping = startColumn == 1
			firstKey = false
			if debug {
				fmt.Printf("parseMapping: first key at column %d, isRootMapping=%v\n", startColumn, isRootMapping)
			}
		} else {
			// Check if we've moved to a different indentation level
			if !isRootMapping && p.currentToken.Column != startColumn {
				if debug {
					fmt.Printf("parseMapping: column changed from %d to %d, breaking\n", startColumn, p.currentToken.Column)
				}
				break
			}
			// For root mappings, only accept keys at column 1
			if isRootMapping && p.currentToken.Column != 1 {
				if debug {
					fmt.Printf("parseMapping: root mapping but column %d != 1, breaking\n", p.currentToken.Column)
				}
				break
			}
		}

		key, err := p.parseKey()
		if err != nil {
			if debug {
				fmt.Printf("parseMapping: parseKey error: %v, currentToken = %v\n", err, p.currentToken)
			}
			break
		}

		p.skipNewlines()

		if p.currentToken.Type != lexer.TokenKey {
			break
		}
		p.advance()

		p.skipNewlines()
		p.collectComments()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		// Check for inline comment after value
		if p.currentToken.Type == lexer.TokenComment {
			if value != nil {
				comment := value.GetComment()
				comment.LineComment = p.currentToken.Value
				value.SetComment(comment)
			}
			p.advance()
		}

		entry := &ast.MappingEntry{
			Key:   key,
			Value: value,
		}

		mapping.Content = append(mapping.Content, entry)
		p.skipNewlines()
		if debug {
			fmt.Printf("parseMapping: after entry, currentToken = %v\n", p.currentToken)
		}
	}

	if debug {
		fmt.Printf("parseMapping: returning, currentToken = %v\n", p.currentToken)
	}
	return mapping, nil
}

func (p *Parser) parseFlowMapping() (ast.Node, error) {
	mapping := ast.NewMapping()
	mapping.Style = ast.FlowStyle
	p.attachComments(mapping)
	p.advance()

	for p.currentToken.Type != lexer.TokenFlowMappingEnd {
		p.skipNewlines()
		p.collectComments()

		if p.currentToken.Type == lexer.TokenFlowMappingEnd {
			break
		}

		key, err := p.parseKey()
		if err != nil {
			return nil, err
		}

		p.skipNewlines()

		if p.currentToken.Type != lexer.TokenKey {
			return nil, fmt.Errorf("expected ':', got %s", p.currentToken.Type)
		}
		p.advance()

		p.skipNewlines()
		p.collectComments()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		entry := &ast.MappingEntry{
			Key:   key,
			Value: value,
		}

		mapping.Content = append(mapping.Content, entry)

		p.skipNewlines()

		if p.currentToken.Type == lexer.TokenFlowEntry {
			p.advance()
			p.skipNewlines()
		}
	}

	if p.currentToken.Type == lexer.TokenFlowMappingEnd {
		p.advance()
	}

	return mapping, nil
}

func (p *Parser) parseKey() (ast.Node, error) {
	if p.currentToken.Type == lexer.TokenString || p.currentToken.Type == lexer.TokenNumber ||
		p.currentToken.Type == lexer.TokenBoolean || p.currentToken.Type == lexer.TokenNull {
		node := ast.NewScalar(p.currentToken.Value)
		p.attachComments(node)
		p.advance()
		return node, nil
	}
	return nil, fmt.Errorf("expected key, got %s", p.currentToken.Type)
}

func (p *Parser) parseNumber() ast.Node {
	value := p.currentToken.Value
	node := ast.NewScalar(value)

	if strings.Contains(value, ".") || strings.Contains(value, "e") || strings.Contains(value, "E") ||
		value == ".inf" || value == "-.inf" || value == ".nan" {
		node.SetTag("!!float")
	} else {
		node.SetTag("!!int")
	}

	return node
}

func (p *Parser) isMapping() bool {
	if debug {
		fmt.Printf("isMapping: currentToken = %v\n", p.currentToken)
	}
	if p.currentToken.Type != lexer.TokenString && p.currentToken.Type != lexer.TokenNumber &&
		p.currentToken.Type != lexer.TokenBoolean {
		if debug {
			fmt.Printf("isMapping: not a valid key type, returning false\n")
		}
		return false
	}

	nextToken, err := p.scanner.Scan()
	if err != nil {
		if debug {
			fmt.Printf("isMapping: scan error: %v\n", err)
		}
		return false
	}

	isKey := nextToken.Type == lexer.TokenKey
	if debug {
		fmt.Printf("isMapping: nextToken = %v, isKey = %v\n", nextToken, isKey)
	}
	p.scanner.PushBack(nextToken)
	return isKey
}

func (p *Parser) advance() {
	token, err := p.scanner.Scan()
	if err != nil {
		p.currentToken = lexer.Token{Type: lexer.TokenError, Value: err.Error()}
		return
	}
	p.currentToken = token
}

func (p *Parser) skipNewlines() {
	for p.currentToken.Type == lexer.TokenNewLine {
		p.advance()
	}
}

func (p *Parser) collectComments() {
	for p.currentToken.Type == lexer.TokenComment {
		p.comments = append(p.comments, p.currentToken)
		p.advance()
		// Only skip newlines if we're collecting head comments
		if p.currentToken.Type == lexer.TokenNewLine {
			p.skipNewlines()
		}
	}
}

func (p *Parser) attachComments(node ast.Node) {
	if len(p.comments) > 0 {
		comment := ast.Comment{}
		for _, c := range p.comments {
			comment.HeadComment += c.Value + "\n"
		}
		node.SetComment(comment)
		p.comments = p.comments[:0]
	}
}

func Parse(data []byte) (ast.Node, error) {
	return ParseReader(bytes.NewReader(data))
}

func ParseReader(r io.Reader) (ast.Node, error) {
	parser := NewParser(r)
	return parser.Parse()
}
