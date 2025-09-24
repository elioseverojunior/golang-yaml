package yaml

import (
	"bytes"
	"io"

	"golang-yaml/v1/ast"
	"golang-yaml/v1/parser"
)

type Marshaler interface {
	MarshalYAML() (interface{}, error)
}

type Unmarshaler interface {
	UnmarshalYAML(value interface{}) error
}

func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.Encode(v)
	return buf.Bytes(), err
}

func MarshalNode(node ast.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.EncodeNode(node)
	return buf.Bytes(), err
}

func Unmarshal(data []byte, v interface{}) error {
	return UnmarshalReader(bytes.NewReader(data), v)
}

func UnmarshalReader(r io.Reader, v interface{}) error {
	dec := NewDecoder(r)
	return dec.Decode(v)
}

func UnmarshalNode(data []byte) (ast.Node, error) {
	return parser.Parse(data)
}

func UnmarshalNodeReader(r io.Reader) (ast.Node, error) {
	return parser.ParseReader(r)
}
