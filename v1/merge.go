package yaml

import (
	"fmt"
	"reflect"

	"golang-yaml/v1/ast"
)

type MergeMode int

const (
	MergeOverride MergeMode = iota
	MergePreserve
	MergeDeep
	MergeAppend
)

type MergeOptions struct {
	Mode               MergeMode
	ArrayMergeStrategy ArrayMergeStrategy
	PreserveComments   bool
	PreserveOrder      bool
	AllowTypeMismatch  bool
	CustomMergeFunc    func(path string, a, b interface{}) (interface{}, error)
}

type ArrayMergeStrategy int

const (
	ArrayReplace ArrayMergeStrategy = iota
	ArrayAppend
	ArrayMergeByIndex
	ArrayMergeByKey
	ArrayUnion
)

func Merge(a, b []byte, opts ...MergeOptions) ([]byte, error) {
	nodeA, err := UnmarshalNode(a)
	if err != nil {
		return nil, fmt.Errorf("failed to parse first document: %w", err)
	}

	nodeB, err := UnmarshalNode(b)
	if err != nil {
		return nil, fmt.Errorf("failed to parse second document: %w", err)
	}

	options := MergeOptions{
		Mode:               MergeDeep,
		ArrayMergeStrategy: ArrayReplace,
		PreserveComments:   true,
		PreserveOrder:      false,
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	merged, err := MergeNodes(nodeA, nodeB, options)
	if err != nil {
		return nil, err
	}

	return MarshalNode(merged)
}

func MergeNodes(a, b ast.Node, opts MergeOptions) (ast.Node, error) {
	return mergeNodesRecursive(a, b, opts, "")
}

func mergeNodesRecursive(a, b ast.Node, opts MergeOptions, path string) (ast.Node, error) {
	if opts.CustomMergeFunc != nil {
		result, err := opts.CustomMergeFunc(path, nodeToInterface(a), nodeToInterface(b))
		if err == nil && result != nil {
			return interfaceToNode(result)
		}
	}

	if a == nil {
		return b.Clone(), nil
	}
	if b == nil {
		return a.Clone(), nil
	}

	if a.Kind() != b.Kind() {
		if !opts.AllowTypeMismatch {
			return nil, fmt.Errorf("type mismatch at %s: %v vs %v", path, a.Kind(), b.Kind())
		}
		if opts.Mode == MergeOverride {
			return b.Clone(), nil
		}
		return a.Clone(), nil
	}

	switch a.Kind() {
	case ast.DocumentNode:
		return mergeDocuments(a.(*ast.Document), b.(*ast.Document), opts, path)
	case ast.MappingNode:
		return mergeMappings(a.(*ast.Mapping), b.(*ast.Mapping), opts, path)
	case ast.SequenceNode:
		return mergeSequences(a.(*ast.Sequence), b.(*ast.Sequence), opts, path)
	case ast.ScalarNode:
		return mergeScalars(a.(*ast.Scalar), b.(*ast.Scalar), opts, path)
	default:
		if opts.Mode == MergeOverride {
			return b.Clone(), nil
		}
		return a.Clone(), nil
	}
}

func mergeDocuments(a, b *ast.Document, opts MergeOptions, path string) (ast.Node, error) {
	merged := &ast.Document{
		Content: make([]ast.Node, 0),
	}

	if opts.PreserveComments {
		merged.SetComment(mergeComments(a.GetComment(), b.GetComment()))
	}

	if len(a.Content) == 0 {
		merged.Content = cloneNodes(b.Content)
	} else if len(b.Content) == 0 {
		merged.Content = cloneNodes(a.Content)
	} else {
		for i := 0; i < len(a.Content) && i < len(b.Content); i++ {
			node, err := mergeNodesRecursive(a.Content[i], b.Content[i], opts, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			merged.Content = append(merged.Content, node)
		}

		if len(b.Content) > len(a.Content) {
			for i := len(a.Content); i < len(b.Content); i++ {
				merged.Content = append(merged.Content, b.Content[i].Clone())
			}
		} else if opts.Mode == MergePreserve && len(a.Content) > len(b.Content) {
			for i := len(b.Content); i < len(a.Content); i++ {
				merged.Content = append(merged.Content, a.Content[i].Clone())
			}
		}
	}

	return merged, nil
}

func mergeMappings(a, b *ast.Mapping, opts MergeOptions, path string) (ast.Node, error) {
	merged := &ast.Mapping{
		Content: make([]*ast.MappingEntry, 0),
		Style:   a.Style,
	}

	if opts.PreserveComments {
		merged.SetComment(mergeComments(a.GetComment(), b.GetComment()))
	}

	aMap := make(map[string]*ast.MappingEntry)
	aKeys := make([]string, 0)
	for _, entry := range a.Content {
		key := getNodeStringValue(entry.Key)
		aMap[key] = entry
		aKeys = append(aKeys, key)
	}

	bMap := make(map[string]*ast.MappingEntry)
	bKeys := make([]string, 0)
	for _, entry := range b.Content {
		key := getNodeStringValue(entry.Key)
		bMap[key] = entry
		bKeys = append(bKeys, key)
	}

	processedKeys := make(map[string]bool)

	keys := aKeys
	if !opts.PreserveOrder {
		for _, key := range bKeys {
			if _, exists := aMap[key]; !exists {
				keys = append(keys, key)
			}
		}
	} else {
		keys = mergeKeyOrder(aKeys, bKeys)
	}

	for _, key := range keys {
		if processedKeys[key] {
			continue
		}
		processedKeys[key] = true

		aEntry := aMap[key]
		bEntry := bMap[key]

		if aEntry == nil && bEntry != nil {
			merged.Content = append(merged.Content, cloneEntry(bEntry))
		} else if aEntry != nil && bEntry == nil {
			if opts.Mode != MergeOverride {
				merged.Content = append(merged.Content, cloneEntry(aEntry))
			}
		} else if aEntry != nil && bEntry != nil {
			mergedValue, err := mergeNodesRecursive(
				aEntry.Value,
				bEntry.Value,
				opts,
				fmt.Sprintf("%s.%s", path, key),
			)
			if err != nil {
				return nil, err
			}

			entry := &ast.MappingEntry{
				Key:   aEntry.Key.Clone(),
				Value: mergedValue,
			}

			if opts.PreserveComments {
				entry.Comment = mergeComments(aEntry.Comment, bEntry.Comment)
			}

			merged.Content = append(merged.Content, entry)
		}
	}

	if opts.Mode != MergePreserve {
		for _, key := range bKeys {
			if !processedKeys[key] {
				merged.Content = append(merged.Content, cloneEntry(bMap[key]))
			}
		}
	}

	return merged, nil
}

func mergeSequences(a, b *ast.Sequence, opts MergeOptions, path string) (ast.Node, error) {
	merged := &ast.Sequence{
		Style: a.Style,
	}

	if opts.PreserveComments {
		merged.SetComment(mergeComments(a.GetComment(), b.GetComment()))
	}

	switch opts.ArrayMergeStrategy {
	case ArrayReplace:
		merged.Content = cloneNodes(b.Content)

	case ArrayAppend:
		merged.Content = append(cloneNodes(a.Content), cloneNodes(b.Content)...)

	case ArrayMergeByIndex:
		maxLen := len(a.Content)
		if len(b.Content) > maxLen {
			maxLen = len(b.Content)
		}

		for i := 0; i < maxLen; i++ {
			var node ast.Node
			var err error

			if i < len(a.Content) && i < len(b.Content) {
				node, err = mergeNodesRecursive(
					a.Content[i],
					b.Content[i],
					opts,
					fmt.Sprintf("%s[%d]", path, i),
				)
				if err != nil {
					return nil, err
				}
			} else if i < len(a.Content) {
				node = a.Content[i].Clone()
			} else {
				node = b.Content[i].Clone()
			}

			merged.Content = append(merged.Content, node)
		}

	case ArrayUnion:
		seen := make(map[string]bool)
		for _, item := range a.Content {
			key := nodeToString(item)
			if !seen[key] {
				merged.Content = append(merged.Content, item.Clone())
				seen[key] = true
			}
		}
		for _, item := range b.Content {
			key := nodeToString(item)
			if !seen[key] {
				merged.Content = append(merged.Content, item.Clone())
				seen[key] = true
			}
		}

	default:
		merged.Content = cloneNodes(b.Content)
	}

	return merged, nil
}

func mergeScalars(a, b *ast.Scalar, opts MergeOptions, path string) (ast.Node, error) {
	if opts.Mode == MergeOverride || opts.Mode == MergeDeep {
		merged := b.Clone().(*ast.Scalar)
		if opts.PreserveComments {
			merged.SetComment(mergeComments(a.GetComment(), b.GetComment()))
		}
		return merged, nil
	}

	merged := a.Clone().(*ast.Scalar)
	if opts.PreserveComments && b.GetComment().HeadComment != "" || b.GetComment().LineComment != "" {
		merged.SetComment(mergeComments(a.GetComment(), b.GetComment()))
	}
	return merged, nil
}

func mergeComments(a, b ast.Comment) ast.Comment {
	merged := ast.Comment{}

	if b.HeadComment != "" {
		merged.HeadComment = b.HeadComment
	} else if a.HeadComment != "" {
		merged.HeadComment = a.HeadComment
	}

	if b.LineComment != "" {
		merged.LineComment = b.LineComment
	} else if a.LineComment != "" {
		merged.LineComment = a.LineComment
	}

	if b.FootComment != "" {
		merged.FootComment = b.FootComment
	} else if a.FootComment != "" {
		merged.FootComment = a.FootComment
	}

	if b.KeyComment != "" {
		merged.KeyComment = b.KeyComment
	} else if a.KeyComment != "" {
		merged.KeyComment = a.KeyComment
	}

	if b.ValueComment != "" {
		merged.ValueComment = b.ValueComment
	} else if a.ValueComment != "" {
		merged.ValueComment = a.ValueComment
	}

	return merged
}

func mergeKeyOrder(aKeys, bKeys []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, key := range aKeys {
		result = append(result, key)
		seen[key] = true
	}

	for _, key := range bKeys {
		if !seen[key] {
			result = append(result, key)
		}
	}

	return result
}

func cloneNodes(nodes []ast.Node) []ast.Node {
	cloned := make([]ast.Node, len(nodes))
	for i, node := range nodes {
		if node != nil {
			cloned[i] = node.Clone()
		}
	}
	return cloned
}

func cloneEntry(entry *ast.MappingEntry) *ast.MappingEntry {
	if entry == nil {
		return nil
	}
	return &ast.MappingEntry{
		Key:     entry.Key.Clone(),
		Value:   entry.Value.Clone(),
		Comment: entry.Comment,
	}
}

func nodeToString(node ast.Node) string {
	data, _ := MarshalNode(node)
	return string(data)
}

func interfaceToNode(v interface{}) (ast.Node, error) {
	enc := NewEncoder(nil)
	return enc.valueToNode(reflect.ValueOf(v))
}

func MergeValue(a, b interface{}, opts ...MergeOptions) (interface{}, error) {
	aBytes, err := Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal first value: %w", err)
	}

	bBytes, err := Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal second value: %w", err)
	}

	merged, err := Merge(aBytes, bBytes, opts...)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := Unmarshal(merged, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged result: %w", err)
	}

	return result, nil
}

func Patch(base []byte, patches ...[]byte) ([]byte, error) {
	result := base
	opts := MergeOptions{
		Mode:               MergeOverride,
		ArrayMergeStrategy: ArrayReplace,
		PreserveComments:   true,
	}

	for i, patch := range patches {
		merged, err := Merge(result, patch, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to apply patch %d: %w", i+1, err)
		}
		result = merged
	}

	return result, nil
}
