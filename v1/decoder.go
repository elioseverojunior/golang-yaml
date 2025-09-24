package yaml

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"

	"golang-yaml/v1/ast"
	"golang-yaml/v1/parser"
)

type Decoder struct {
	reader io.Reader
	strict bool
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{reader: r}
}

func (d *Decoder) SetStrict(strict bool) {
	d.strict = strict
}

func (d *Decoder) Decode(v interface{}) error {
	node, err := parser.ParseReader(d.reader)
	if err != nil {
		return err
	}

	return d.decodeNode(node, reflect.ValueOf(v))
}

func (d *Decoder) decodeNode(node ast.Node, v reflect.Value) error {
	if !v.IsValid() {
		return fmt.Errorf("cannot decode into invalid value")
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return d.decodeNode(node, v.Elem())
	}

	if node == nil {
		v.Set(reflect.Zero(v.Type()))
		return nil
	}

	if v.CanInterface() {
		if unmarshaler, ok := v.Interface().(Unmarshaler); ok {
			value := nodeToInterface(node)
			return unmarshaler.UnmarshalYAML(value)
		}
	}

	switch node.Kind() {
	case ast.DocumentNode:
		doc := node.(*ast.Document)
		if len(doc.Content) == 0 {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		return d.decodeNode(doc.Content[0], v)

	case ast.ScalarNode:
		return d.decodeScalar(node.(*ast.Scalar), v)

	case ast.MappingNode:
		return d.decodeMapping(node.(*ast.Mapping), v)

	case ast.SequenceNode:
		return d.decodeSequence(node.(*ast.Sequence), v)

	case ast.AliasNode:
		return fmt.Errorf("alias nodes should be resolved before decoding")

	default:
		return fmt.Errorf("unknown node kind: %v", node.Kind())
	}
}

func (d *Decoder) decodeScalar(scalar *ast.Scalar, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Interface:
		if v.NumMethod() == 0 {
			value := parseScalarValue(scalar)
			if value == nil {
				v.Set(reflect.Zero(v.Type()))
			} else {
				v.Set(reflect.ValueOf(value))
			}
		}
		return nil

	case reflect.String:
		v.SetString(scalar.Value)
		return nil

	case reflect.Bool:
		b, err := parseBool(scalar.Value)
		if err != nil {
			return err
		}
		v.SetBool(b)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := parseInt(scalar.Value, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := parseUint(scalar.Value, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(u)
		return nil

	case reflect.Float32, reflect.Float64:
		f, err := parseFloat(scalar.Value, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
		return nil

	default:
		return fmt.Errorf("cannot decode scalar into %s", v.Kind())
	}
}

func (d *Decoder) decodeMapping(mapping *ast.Mapping, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Interface:
		if v.NumMethod() == 0 {
			mapValue := make(map[string]interface{})
			for _, entry := range mapping.Content {
				key := getNodeStringValue(entry.Key)
				value := nodeToInterface(entry.Value)
				mapValue[key] = value
			}
			v.Set(reflect.ValueOf(mapValue))
		}
		return nil

	case reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		for _, entry := range mapping.Content {
			keyValue := reflect.New(v.Type().Key()).Elem()
			if err := d.decodeNode(entry.Key, keyValue); err != nil {
				return err
			}

			elemValue := reflect.New(v.Type().Elem()).Elem()
			if err := d.decodeNode(entry.Value, elemValue); err != nil {
				return err
			}

			v.SetMapIndex(keyValue, elemValue)
		}
		return nil

	case reflect.Struct:
		return d.decodeStruct(mapping, v)

	default:
		return fmt.Errorf("cannot decode mapping into %s", v.Kind())
	}
}

func (d *Decoder) decodeStruct(mapping *ast.Mapping, v reflect.Value) error {
	t := v.Type()
	fields := make(map[string]int)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name := field.Name
		tag := field.Tag.Get("yaml")
		if tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			if parts[0] == "-" {
				continue
			}
		}

		fields[strings.ToLower(name)] = i
		if tag != "" && tag != "-" {
			fields[strings.Split(tag, ",")[0]] = i
		}
	}

	for _, entry := range mapping.Content {
		key := getNodeStringValue(entry.Key)

		fieldIndex, ok := fields[strings.ToLower(key)]
		if !ok {
			fieldIndex, ok = fields[key]
		}

		if !ok {
			if d.strict {
				return fmt.Errorf("field %s not found in struct", key)
			}
			continue
		}

		field := v.Field(fieldIndex)
		if err := d.decodeNode(entry.Value, field); err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) decodeSequence(sequence *ast.Sequence, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Interface:
		if v.NumMethod() == 0 {
			slice := make([]interface{}, len(sequence.Content))
			for i, item := range sequence.Content {
				slice[i] = nodeToInterface(item)
			}
			v.Set(reflect.ValueOf(slice))
		}
		return nil

	case reflect.Slice:
		slice := reflect.MakeSlice(v.Type(), len(sequence.Content), len(sequence.Content))
		for i, item := range sequence.Content {
			if err := d.decodeNode(item, slice.Index(i)); err != nil {
				return err
			}
		}
		v.Set(slice)
		return nil

	case reflect.Array:
		if v.Len() < len(sequence.Content) {
			return fmt.Errorf("array too small for sequence")
		}
		for i, item := range sequence.Content {
			if err := d.decodeNode(item, v.Index(i)); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("cannot decode sequence into %s", v.Kind())
	}
}

func getNodeStringValue(node ast.Node) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *ast.Scalar:
		return n.Value
	default:
		return ""
	}
}

func nodeToInterface(node ast.Node) interface{} {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Scalar:
		return parseScalarValue(n)

	case *ast.Mapping:
		m := make(map[string]interface{})
		for _, entry := range n.Content {
			key := getNodeStringValue(entry.Key)
			m[key] = nodeToInterface(entry.Value)
		}
		return m

	case *ast.Sequence:
		s := make([]interface{}, len(n.Content))
		for i, item := range n.Content {
			s[i] = nodeToInterface(item)
		}
		return s

	case *ast.Document:
		if len(n.Content) == 0 {
			return nil
		}
		return nodeToInterface(n.Content[0])

	default:
		return nil
	}
}

func parseScalarValue(scalar *ast.Scalar) interface{} {
	value := scalar.Value
	tag := scalar.Tag()

	if tag == "!!null" || value == "" || value == "null" || value == "~" {
		return nil
	}

	if tag == "!!bool" {
		if b, err := parseBool(value); err == nil {
			return b
		}
	}

	if tag == "!!int" {
		if i, err := parseInt(value, 64); err == nil {
			return i
		}
	}

	if tag == "!!float" {
		if f, err := parseFloat(value, 64); err == nil {
			return f
		}
	}

	if tag == "!!str" {
		return value
	}

	if b, err := parseBool(value); err == nil {
		return b
	}

	if i, err := parseInt(value, 64); err == nil {
		return i
	}

	if f, err := parseFloat(value, 64); err == nil {
		return f
	}

	return value
}

func parseBool(value string) (bool, error) {
	lower := strings.ToLower(value)
	switch lower {
	case "true", "yes", "on":
		return true, nil
	case "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean value: %s", value)
}

func parseInt(value string, bitSize int) (int64, error) {
	value = strings.ReplaceAll(value, "_", "")

	if strings.HasPrefix(value, "0x") {
		return strconv.ParseInt(value[2:], 16, bitSize)
	}
	if strings.HasPrefix(value, "0o") {
		return strconv.ParseInt(value[2:], 8, bitSize)
	}
	if strings.HasPrefix(value, "0b") {
		return strconv.ParseInt(value[2:], 2, bitSize)
	}

	return strconv.ParseInt(value, 10, bitSize)
}

func parseUint(value string, bitSize int) (uint64, error) {
	value = strings.ReplaceAll(value, "_", "")

	if strings.HasPrefix(value, "0x") {
		return strconv.ParseUint(value[2:], 16, bitSize)
	}
	if strings.HasPrefix(value, "0o") {
		return strconv.ParseUint(value[2:], 8, bitSize)
	}
	if strings.HasPrefix(value, "0b") {
		return strconv.ParseUint(value[2:], 2, bitSize)
	}

	return strconv.ParseUint(value, 10, bitSize)
}

func parseFloat(value string, bitSize int) (float64, error) {
	value = strings.ReplaceAll(value, "_", "")

	switch value {
	case ".inf", "+.inf":
		return math.Inf(1), nil
	case "-.inf":
		return math.Inf(-1), nil
	case ".nan":
		return math.NaN(), nil
	}

	return strconv.ParseFloat(value, bitSize)
}
