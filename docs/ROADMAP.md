# golang-yaml Roadmap

## Phase 1: Core Implementation (Current)
- [x] Project setup and structure
- [x] Basic tokenizer/scanner implementation (v1/lexer)
- [x] Core parser for basic YAML structures (v1/parser)
- [x] Simple encoder/decoder for Go types (v1/decoder.go, v1/encoder.go)
- [x] Basic test suite (v1/yaml_test.go)

## Phase 2: Full YAML 1.2.2 Support
- [ ] Complete all YAML 1.2.2 features
- [ ] Anchor and alias support
- [ ] Tag support
- [ ] Multi-document support
- [ ] All scalar styles (literal, folded, etc.)
- [ ] Comprehensive test coverage

## Phase 3: Advanced Features
- [ ] Custom tag handlers
- [ ] Stream processing for large files
- [ ] Schema validation
- [ ] JSON compatibility mode
- [ ] Performance optimizations

## Phase 4: Tooling and Ecosystem
- [ ] YAML linter
- [ ] YAML formatter
- [ ] Conversion tools (YAML to JSON, etc.)
- [ ] Documentation generator
- [ ] VS Code extension support

## Design Principles
1. **Clean API**: Simple and intuitive API similar to encoding/json
2. **Performance**: Efficient parsing and encoding
3. **Correctness**: Full YAML 1.2.2 compliance
4. **Error Handling**: Clear and helpful error messages
5. **Extensibility**: Support for custom types and tags
6. **Zero Dependencies**: No external dependencies beyond Go standard library