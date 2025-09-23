# YAML 1.2.2 Compliance

This document outlines the YAML 1.2.2 specification compliance for the goyaml-yaml library.

## Core Features

### Basic Types
- [ ] Null values (null, ~, empty)
- [ ] Boolean values (true/false, yes/no, on/off)
- [ ] Integer values (decimal, octal 0o, hexadecimal 0x, binary 0b)
- [ ] Float values (including .inf, -.inf, .nan)
- [ ] String values (plain, single-quoted, double-quoted, literal |, folded >)

### Collections
- [ ] Sequences (arrays/lists)
- [ ] Mappings (maps/dictionaries)
- [ ] Nested structures
- [ ] Flow style collections
- [ ] Block style collections

### Advanced Features
- [ ] Anchors (&) and Aliases (*)
- [ ] Tags (!! and !)
- [ ] Multi-document streams (---)
- [ ] Directives (%YAML, %TAG)
- [ ] Comments (#)
- [ ] Merge keys (<<)

### String Styles
- [ ] Plain scalars
- [ ] Single-quoted scalars
- [ ] Double-quoted scalars with escape sequences
- [ ] Literal block scalars (|)
- [ ] Folded block scalars (>)
- [ ] Block chomping indicators (+, -)
- [ ] Block indentation indicators

### Schema Support
- [ ] Failsafe schema
- [ ] JSON schema
- [ ] Core schema (default)

## Compliance Level

The library aims for full YAML 1.2.2 compliance with focus on:
1. Correct parsing of all YAML 1.2.2 constructs
2. Proper error reporting with line/column information
3. Round-trip preservation where possible
4. Performance optimization without sacrificing correctness