package lexer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Scanner struct {
	reader      *bufio.Reader
	buffer      []byte
	position    int
	line        int
	column      int
	offset      int
	indentStack []int
	inFlow      int
	tokens      []Token
	tokenIndex  int
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		reader:      bufio.NewReader(r),
		line:        1,
		column:      1,
		indentStack: []int{0},
	}
}

func (s *Scanner) Scan() (Token, error) {
	if s.tokenIndex < len(s.tokens) {
		token := s.tokens[s.tokenIndex]
		s.tokenIndex++
		return token, nil
	}

	return s.scanNext()
}

func (s *Scanner) PushBack(token Token) {
	s.tokens = append([]Token{token}, s.tokens[s.tokenIndex:]...)
	s.tokenIndex = 0
}

func (s *Scanner) scanNext() (Token, error) {
	s.skipWhitespace()

	if s.isEOF() {
		return s.makeToken(TokenEOF, ""), nil
	}

	ch := s.peek()

	if ch == '#' {
		return s.scanComment()
	}

	if ch == '\n' {
		return s.scanNewline()
	}

	if s.column == 1 && ch == '-' && s.peekAhead(1) == '-' && s.peekAhead(2) == '-' {
		return s.scanDocumentStart()
	}

	if s.column == 1 && ch == '.' && s.peekAhead(1) == '.' && s.peekAhead(2) == '.' {
		return s.scanDocumentEnd()
	}

	if ch == '-' && s.peekAhead(1) == ' ' {
		return s.scanSequenceItem()
	}

	if ch == '[' {
		return s.scanFlowSequenceStart()
	}

	if ch == ']' {
		return s.scanFlowSequenceEnd()
	}

	if ch == '{' {
		return s.scanFlowMappingStart()
	}

	if ch == '}' {
		return s.scanFlowMappingEnd()
	}

	if ch == ',' && s.inFlow > 0 {
		return s.scanFlowEntry()
	}

	if ch == ':' && (s.peekAhead(1) == ' ' || s.peekAhead(1) == '\n' || s.isEOFAt(1)) {
		return s.scanKey()
	}

	if ch == '|' {
		return s.scanLiteralBlock()
	}

	if ch == '>' {
		return s.scanFoldedBlock()
	}

	if ch == '\'' {
		return s.scanSingleQuotedString()
	}

	if ch == '"' {
		return s.scanDoubleQuotedString()
	}

	if ch == '&' {
		return s.scanAnchor()
	}

	if ch == '*' {
		return s.scanAlias()
	}

	if ch == '!' {
		return s.scanTag()
	}

	return s.scanScalar()
}

func (s *Scanner) scanComment() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	var comment bytes.Buffer
	for !s.isEOF() && s.peek() != '\n' {
		comment.WriteByte(s.peek())
		s.advance()
	}

	return Token{
		Type:   TokenComment,
		Value:  strings.TrimSpace(comment.String()),
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanNewline() (Token, error) {
	token := s.makeToken(TokenNewLine, "\n")
	s.advance()
	s.line++
	s.column = 1
	return token, nil
}

func (s *Scanner) scanDocumentStart() (Token, error) {
	token := s.makeToken(TokenDocumentStart, "---")
	s.advance()
	s.advance()
	s.advance()
	return token, nil
}

func (s *Scanner) scanDocumentEnd() (Token, error) {
	token := s.makeToken(TokenDocumentEnd, "...")
	s.advance()
	s.advance()
	s.advance()
	return token, nil
}

func (s *Scanner) scanSequenceItem() (Token, error) {
	token := s.makeToken(TokenSequenceItem, "-")
	s.advance()
	s.advance()
	return token, nil
}

func (s *Scanner) scanFlowSequenceStart() (Token, error) {
	token := s.makeToken(TokenFlowSequenceStart, "[")
	s.advance()
	s.inFlow++
	return token, nil
}

func (s *Scanner) scanFlowSequenceEnd() (Token, error) {
	token := s.makeToken(TokenFlowSequenceEnd, "]")
	s.advance()
	if s.inFlow > 0 {
		s.inFlow--
	}
	return token, nil
}

func (s *Scanner) scanFlowMappingStart() (Token, error) {
	token := s.makeToken(TokenFlowMappingStart, "{")
	s.advance()
	s.inFlow++
	return token, nil
}

func (s *Scanner) scanFlowMappingEnd() (Token, error) {
	token := s.makeToken(TokenFlowMappingEnd, "}")
	s.advance()
	if s.inFlow > 0 {
		s.inFlow--
	}
	return token, nil
}

func (s *Scanner) scanFlowEntry() (Token, error) {
	token := s.makeToken(TokenFlowEntry, ",")
	s.advance()
	return token, nil
}

func (s *Scanner) scanKey() (Token, error) {
	token := s.makeToken(TokenKey, ":")
	s.advance()
	return token, nil
}

func (s *Scanner) scanLiteralBlock() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	chomping := s.scanChompingIndicator()
	s.skipToEndOfLine()

	if !s.isEOF() && s.peek() == '\n' {
		s.advance()
		s.line++
		s.column = 1
	}

	baseIndent := s.countIndent()
	var content bytes.Buffer

	for !s.isEOF() {
		indent := s.countIndent()
		if indent < baseIndent && s.peek() != '\n' {
			break
		}

		s.skipIndent(indent)

		for !s.isEOF() && s.peek() != '\n' {
			content.WriteByte(s.peek())
			s.advance()
		}

		if !s.isEOF() {
			content.WriteByte('\n')
			s.advance()
			s.line++
			s.column = 1
		}
	}

	value := s.applyChomping(content.String(), chomping)

	return Token{
		Type:   TokenLiteralBlock,
		Value:  value,
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanFoldedBlock() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	chomping := s.scanChompingIndicator()
	s.skipToEndOfLine()

	if !s.isEOF() && s.peek() == '\n' {
		s.advance()
		s.line++
		s.column = 1
	}

	baseIndent := s.countIndent()
	var content bytes.Buffer
	lastWasEmpty := false

	for !s.isEOF() {
		indent := s.countIndent()
		if indent < baseIndent && s.peek() != '\n' {
			break
		}

		s.skipIndent(indent)

		lineEmpty := s.peek() == '\n'

		if !lineEmpty {
			if content.Len() > 0 && !lastWasEmpty {
				content.WriteByte(' ')
			}

			for !s.isEOF() && s.peek() != '\n' {
				content.WriteByte(s.peek())
				s.advance()
			}
			lastWasEmpty = false
		} else {
			if content.Len() > 0 {
				content.WriteByte('\n')
			}
			lastWasEmpty = true
		}

		if !s.isEOF() && s.peek() == '\n' {
			s.advance()
			s.line++
			s.column = 1
		}
	}

	value := s.applyChomping(content.String(), chomping)

	return Token{
		Type:   TokenFoldedBlock,
		Value:  value,
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanChompingIndicator() string {
	if ch := s.peek(); ch == '+' || ch == '-' {
		s.advance()
		return string(ch)
	}
	return ""
}

func (s *Scanner) applyChomping(value, chomping string) string {
	if chomping == "-" {
		return strings.TrimRight(value, "\n")
	} else if chomping != "+" {
		value = strings.TrimRight(value, "\n") + "\n"
	}
	return value
}

func (s *Scanner) scanSingleQuotedString() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	var str bytes.Buffer
	for !s.isEOF() {
		ch := s.peek()
		if ch == '\'' {
			if s.peekAhead(1) == '\'' {
				str.WriteByte('\'')
				s.advance()
				s.advance()
			} else {
				s.advance()
				break
			}
		} else {
			str.WriteByte(ch)
			s.advance()
			if ch == '\n' {
				s.line++
				s.column = 1
			}
		}
	}

	return Token{
		Type:   TokenString,
		Value:  str.String(),
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanDoubleQuotedString() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	var str bytes.Buffer
	for !s.isEOF() {
		ch := s.peek()
		if ch == '"' {
			s.advance()
			break
		} else if ch == '\\' {
			s.advance()
			if !s.isEOF() {
				escape := s.peek()
				s.advance()
				switch escape {
				case 'n':
					str.WriteByte('\n')
				case 't':
					str.WriteByte('\t')
				case 'r':
					str.WriteByte('\r')
				case '\\':
					str.WriteByte('\\')
				case '"':
					str.WriteByte('"')
				case '0':
					str.WriteByte('\x00')
				case 'a':
					str.WriteByte('\a')
				case 'b':
					str.WriteByte('\b')
				case 'v':
					str.WriteByte('\v')
				case 'f':
					str.WriteByte('\f')
				case 'e':
					str.WriteByte('\x1b')
				default:
					str.WriteByte(escape)
				}
			}
		} else {
			str.WriteByte(ch)
			s.advance()
			if ch == '\n' {
				s.line++
				s.column = 1
			}
		}
	}

	return Token{
		Type:   TokenString,
		Value:  str.String(),
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanAnchor() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	var anchor bytes.Buffer
	for !s.isEOF() && s.isAnchorChar(s.peek()) {
		anchor.WriteByte(s.peek())
		s.advance()
	}

	return Token{
		Type:   TokenAnchor,
		Value:  anchor.String(),
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanAlias() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	var alias bytes.Buffer
	for !s.isEOF() && s.isAnchorChar(s.peek()) {
		alias.WriteByte(s.peek())
		s.advance()
	}

	return Token{
		Type:   TokenAlias,
		Value:  alias.String(),
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanTag() (Token, error) {
	startPos := s.makePosition()
	s.advance()

	var tag bytes.Buffer
	if s.peek() == '!' {
		tag.WriteByte('!')
		s.advance()
	}

	for !s.isEOF() && !unicode.IsSpace(rune(s.peek())) {
		tag.WriteByte(s.peek())
		s.advance()
	}

	return Token{
		Type:   TokenTag,
		Value:  tag.String(),
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) scanScalar() (Token, error) {
	startPos := s.makePosition()

	var scalar bytes.Buffer
	for !s.isEOF() {
		ch := s.peek()
		if ch == ':' && (s.peekAhead(1) == ' ' || s.peekAhead(1) == '\n' || s.isEOFAt(1)) {
			break
		}
		if ch == '\n' || ch == '#' {
			break
		}
		if s.inFlow > 0 && (ch == ',' || ch == '}' || ch == ']') {
			break
		}
		scalar.WriteByte(ch)
		s.advance()
	}

	value := strings.TrimSpace(scalar.String())
	tokenType := s.detectScalarType(value)

	return Token{
		Type:   tokenType,
		Value:  value,
		Line:   startPos.line,
		Column: startPos.column,
		Offset: startPos.offset,
	}, nil
}

func (s *Scanner) detectScalarType(value string) TokenType {
	if value == "null" || value == "~" || value == "" {
		return TokenNull
	}

	lower := strings.ToLower(value)
	if lower == "true" || lower == "false" || lower == "yes" || lower == "no" || lower == "on" || lower == "off" {
		return TokenBoolean
	}

	if s.isNumber(value) {
		return TokenNumber
	}

	return TokenString
}

func (s *Scanner) isNumber(value string) bool {
	if len(value) == 0 {
		return false
	}

	if value == ".inf" || value == "-.inf" || value == "+.inf" || value == ".nan" {
		return true
	}

	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0o") || strings.HasPrefix(value, "0b") {
		return true
	}

	for i, ch := range value {
		if !unicode.IsDigit(ch) && ch != '.' && ch != '-' && ch != '+' && ch != 'e' && ch != 'E' && ch != '_' {
			return false
		}
		if (ch == '-' || ch == '+') && i != 0 && value[i-1] != 'e' && value[i-1] != 'E' {
			return false
		}
	}

	return true
}

func (s *Scanner) isAnchorChar(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' || ch == '-'
}

func (s *Scanner) skipWhitespace() {
	for !s.isEOF() && (s.peek() == ' ' || s.peek() == '\t') {
		s.advance()
	}
}

func (s *Scanner) skipToEndOfLine() {
	for !s.isEOF() && s.peek() != '\n' {
		s.advance()
	}
}

func (s *Scanner) countIndent() int {
	indent := 0
	pos := s.position
	for pos < len(s.buffer) && s.buffer[pos] == ' ' {
		indent++
		pos++
	}
	return indent
}

func (s *Scanner) skipIndent(count int) {
	for i := 0; i < count && !s.isEOF() && s.peek() == ' '; i++ {
		s.advance()
	}
}

type scannerPosition struct {
	line   int
	column int
	offset int
}

func (s *Scanner) makePosition() scannerPosition {
	return scannerPosition{
		line:   s.line,
		column: s.column,
		offset: s.offset,
	}
}

func (s *Scanner) makeToken(t TokenType, value string) Token {
	return Token{
		Type:   t,
		Value:  value,
		Line:   s.line,
		Column: s.column,
		Offset: s.offset,
	}
}

func (s *Scanner) peek() byte {
	if s.position >= len(s.buffer) {
		s.fillBuffer()
	}
	if s.position < len(s.buffer) {
		return s.buffer[s.position]
	}
	return 0
}

func (s *Scanner) peekAhead(offset int) byte {
	for s.position+offset >= len(s.buffer) {
		if !s.fillBuffer() {
			break
		}
	}
	if s.position+offset < len(s.buffer) {
		return s.buffer[s.position+offset]
	}
	return 0
}

func (s *Scanner) advance() {
	if s.position < len(s.buffer) {
		s.position++
		s.column++
		s.offset++
	}
}

func (s *Scanner) isEOF() bool {
	return s.position >= len(s.buffer) && !s.fillBuffer()
}

func (s *Scanner) isEOFAt(offset int) bool {
	for s.position+offset >= len(s.buffer) {
		if !s.fillBuffer() {
			return true
		}
	}
	return false
}

func (s *Scanner) fillBuffer() bool {
	if s.reader == nil {
		return false
	}

	b, err := s.reader.ReadByte()
	if err != nil {
		return false
	}

	s.buffer = append(s.buffer, b)
	return true
}

func (s *Scanner) Error(msg string) error {
	return fmt.Errorf("%s at line %d, column %d", msg, s.line, s.column)
}
