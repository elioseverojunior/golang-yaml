package yaml

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang-yaml/v1/ast"
)

type Encoder struct {
	writer io.Writer
	indent int
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		writer: w,
		indent: 2,
	}
}

func (e *Encoder) SetIndent(spaces int) {
	e.indent = spaces
}

func (e *Encoder) Encode(v interface{}) error {
	node, err := e.valueToNode(reflect.ValueOf(v))
	if err != nil {
		return err
	}
	return e.EncodeNode(node)
}

func (e *Encoder) EncodeNode(node ast.Node) error {
	var buf bytes.Buffer
	if err := e.encodeNode(&buf, node, 0, false); err != nil {
		return err
	}
	_, err := e.writer.Write(buf.Bytes())
	return err
}

func (e *Encoder) valueToNode(v reflect.Value) (ast.Node, error) {
	if !v.IsValid() {
		return ast.NewScalar("null"), nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ast.NewScalar("null"), nil
		}
		return e.valueToNode(v.Elem())
	}

	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ast.NewScalar("null"), nil
		}
		return e.valueToNode(v.Elem())
	}

	if v.CanInterface() {
		if marshaler, ok := v.Interface().(Marshaler); ok {
			value, err := marshaler.MarshalYAML()
			if err != nil {
				return nil, err
			}
			return e.valueToNode(reflect.ValueOf(value))
		}
	}

	switch v.Kind() {
	case reflect.Bool:
		return ast.NewScalar(strconv.FormatBool(v.Bool())), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ast.NewScalar(strconv.FormatInt(v.Int(), 10)), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return ast.NewScalar(strconv.FormatUint(v.Uint(), 10)), nil

	case reflect.Float32, reflect.Float64:
		f := v.Float()
		var s string
		switch {
		case math.IsNaN(f):
			s = ".nan"
		case math.IsInf(f, 1):
			s = ".inf"
		case math.IsInf(f, -1):
			s = "-.inf"
		default:
			s = strconv.FormatFloat(f, 'g', -1, v.Type().Bits())
		}
		return ast.NewScalar(s), nil

	case reflect.String:
		return e.createStringNode(v.String()), nil

	case reflect.Slice, reflect.Array:
		return e.valueToSequence(v)

	case reflect.Map:
		return e.valueToMapping(v)

	case reflect.Struct:
		return e.structToMapping(v)

	default:
		return nil, fmt.Errorf("unsupported type: %s", v.Kind())
	}
}

func (e *Encoder) createStringNode(s string) *ast.Scalar {
	node := ast.NewScalar(s)

	if strings.Contains(s, "\n") {
		if strings.Contains(s, "  ") || strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
			node.Style = ast.LiteralStyle
		} else {
			node.Style = ast.FoldedStyle
		}
	} else if needsQuoting(s) {
		node.Style = ast.DoubleQuotedStyle
	}

	return node
}

func (e *Encoder) valueToSequence(v reflect.Value) (ast.Node, error) {
	sequence := ast.NewSequence()

	for i := 0; i < v.Len(); i++ {
		item, err := e.valueToNode(v.Index(i))
		if err != nil {
			return nil, err
		}
		sequence.Content = append(sequence.Content, item)
	}

	return sequence, nil
}

func (e *Encoder) valueToMapping(v reflect.Value) (ast.Node, error) {
	mapping := ast.NewMapping()

	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i].Interface()) < fmt.Sprintf("%v", keys[j].Interface())
	})

	for _, key := range keys {
		keyNode, err := e.valueToNode(key)
		if err != nil {
			return nil, err
		}

		valueNode, err := e.valueToNode(v.MapIndex(key))
		if err != nil {
			return nil, err
		}

		entry := &ast.MappingEntry{
			Key:   keyNode,
			Value: valueNode,
		}
		mapping.Content = append(mapping.Content, entry)
	}

	return mapping, nil
}

func (e *Encoder) structToMapping(v reflect.Value) (ast.Node, error) {
	mapping := ast.NewMapping()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		fieldValue := v.Field(i)
		if !fieldValue.IsValid() || isZeroValue(fieldValue) {
			if tag := field.Tag.Get("yaml"); strings.Contains(tag, ",omitempty") {
				continue
			}
		}

		name := field.Name
		tag := field.Tag.Get("yaml")
		if tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
		}

		keyNode := ast.NewScalar(name)
		valueNode, err := e.valueToNode(fieldValue)
		if err != nil {
			return nil, err
		}

		entry := &ast.MappingEntry{
			Key:   keyNode,
			Value: valueNode,
		}
		mapping.Content = append(mapping.Content, entry)
	}

	return mapping, nil
}

func (e *Encoder) encodeNode(w io.Writer, node ast.Node, indent int, inline bool) error {
	if node == nil {
		fmt.Fprint(w, "null")
		return nil
	}

	comment := node.GetComment()
	if comment.HeadComment != "" && !inline {
		for _, line := range strings.Split(strings.TrimSpace(comment.HeadComment), "\n") {
			e.writeIndent(w, indent)
			fmt.Fprintf(w, "# %s\n", line)
		}
	}

	switch n := node.(type) {
	case *ast.Document:
		for i, content := range n.Content {
			if i > 0 {
				fmt.Fprintln(w, "\n---")
			}
			if err := e.encodeNode(w, content, indent, false); err != nil {
				return err
			}
		}

	case *ast.Scalar:
		if !inline {
			e.writeIndent(w, indent)
		}
		e.encodeScalar(w, n)

	case *ast.Sequence:
		if err := e.encodeSequence(w, n, indent, inline); err != nil {
			return err
		}

	case *ast.Mapping:
		if err := e.encodeMapping(w, n, indent, inline); err != nil {
			return err
		}

	case *ast.Alias:
		if !inline {
			e.writeIndent(w, indent)
		}
		fmt.Fprintf(w, "*%s", n.Identifier)

	default:
		return fmt.Errorf("unknown node type: %T", node)
	}

	if comment.LineComment != "" {
		fmt.Fprintf(w, " # %s", comment.LineComment)
	}

	if comment.FootComment != "" && !inline {
		fmt.Fprintln(w)
		for _, line := range strings.Split(strings.TrimSpace(comment.FootComment), "\n") {
			e.writeIndent(w, indent)
			fmt.Fprintf(w, "# %s\n", line)
		}
	}

	return nil
}

func (e *Encoder) encodeScalar(w io.Writer, scalar *ast.Scalar) {
	switch scalar.Style {
	case ast.SingleQuotedStyle:
		fmt.Fprintf(w, "'%s'", strings.ReplaceAll(scalar.Value, "'", "''"))
	case ast.DoubleQuotedStyle:
		fmt.Fprintf(w, "%q", scalar.Value)
	case ast.LiteralStyle:
		fmt.Fprint(w, "|")
		if scalar.Value != "" && !strings.HasSuffix(scalar.Value, "\n") {
			fmt.Fprint(w, "-")
		}
		fmt.Fprintln(w)
		for _, line := range strings.Split(scalar.Value, "\n") {
			if line != "" {
				e.writeIndent(w, e.indent)
				fmt.Fprintln(w, line)
			}
		}
	case ast.FoldedStyle:
		fmt.Fprint(w, ">")
		if scalar.Value != "" && !strings.HasSuffix(scalar.Value, "\n") {
			fmt.Fprint(w, "-")
		}
		fmt.Fprintln(w)
		for _, line := range strings.Split(scalar.Value, "\n") {
			if line != "" {
				e.writeIndent(w, e.indent)
				fmt.Fprintln(w, line)
			}
		}
	default:
		fmt.Fprint(w, scalar.Value)
	}
}

func (e *Encoder) encodeSequence(w io.Writer, sequence *ast.Sequence, indent int, inline bool) error {
	if len(sequence.Content) == 0 {
		fmt.Fprint(w, "[]")
		return nil
	}

	if sequence.Style == ast.FlowStyle || inline {
		fmt.Fprint(w, "[")
		for i, item := range sequence.Content {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			if err := e.encodeNode(w, item, 0, true); err != nil {
				return err
			}
		}
		fmt.Fprint(w, "]")
	} else {
		for i, item := range sequence.Content {
			if i > 0 {
				fmt.Fprintln(w)
			}
			e.writeIndent(w, indent)
			fmt.Fprint(w, "- ")

			switch item.(type) {
			case *ast.Mapping, *ast.Sequence:
				fmt.Fprintln(w)
				if err := e.encodeNode(w, item, indent+e.indent, false); err != nil {
					return err
				}
			default:
				var buf bytes.Buffer
				if err := e.encodeNode(&buf, item, 0, true); err != nil {
					return err
				}
				fmt.Fprint(w, strings.TrimSpace(buf.String()))
			}
		}
	}

	return nil
}

func (e *Encoder) encodeMapping(w io.Writer, mapping *ast.Mapping, indent int, inline bool) error {
	if len(mapping.Content) == 0 {
		fmt.Fprint(w, "{}")
		return nil
	}

	if mapping.Style == ast.FlowStyle || inline {
		fmt.Fprint(w, "{")
		for i, entry := range mapping.Content {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			if err := e.encodeNode(w, entry.Key, 0, true); err != nil {
				return err
			}
			fmt.Fprint(w, ": ")
			if err := e.encodeNode(w, entry.Value, 0, true); err != nil {
				return err
			}
		}
		fmt.Fprint(w, "}")
	} else {
		for i, entry := range mapping.Content {
			if i > 0 {
				fmt.Fprintln(w)
			}

			if entry.Comment.KeyComment != "" {
				for _, line := range strings.Split(strings.TrimSpace(entry.Comment.KeyComment), "\n") {
					e.writeIndent(w, indent)
					fmt.Fprintf(w, "# %s\n", line)
				}
			}

			e.writeIndent(w, indent)

			// Write the key
			if err := e.encodeNode(w, entry.Key, 0, true); err != nil {
				return err
			}
			fmt.Fprint(w, ": ")

			// Write the value
			switch entry.Value.(type) {
			case *ast.Mapping, *ast.Sequence:
				fmt.Fprintln(w)
				if err := e.encodeNode(w, entry.Value, indent+e.indent, false); err != nil {
					return err
				}
			default:
				if err := e.encodeNode(w, entry.Value, 0, true); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (e *Encoder) writeIndent(w io.Writer, spaces int) {
	for i := 0; i < spaces; i++ {
		fmt.Fprint(w, " ")
	}
}

func needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	specialValues := []string{
		"true", "false", "yes", "no", "on", "off",
		"null", "~", ".inf", "-.inf", ".nan",
	}

	for _, special := range specialValues {
		if strings.EqualFold(s, special) {
			return true
		}
	}

	if strings.ContainsAny(s, ":#@*&[]{}|>'\"\n\r\t") {
		return true
	}

	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}

	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}

	return false
}

func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
