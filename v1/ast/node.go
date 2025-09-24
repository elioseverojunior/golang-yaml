package ast

import (
	"fmt"
	"sort"
)

type NodeKind int

const (
	DocumentNode NodeKind = iota
	ScalarNode
	MappingNode
	SequenceNode
	AliasNode
)

type Node interface {
	Kind() NodeKind
	Tag() string
	SetTag(tag string)
	GetComment() Comment
	SetComment(comment Comment)
	Position() Position
	SetPosition(pos Position)
	Clone() Node
	String() string
}

type Position struct {
	Line   int
	Column int
	Offset int
}

type Comment struct {
	HeadComment  string
	LineComment  string
	FootComment  string
	KeyComment   string
	ValueComment string
}

type baseNode struct {
	tag     string
	comment Comment
	anchor  string
	pos     Position
}

func (n *baseNode) Tag() string {
	return n.tag
}

func (n *baseNode) SetTag(tag string) {
	n.tag = tag
}

func (n *baseNode) GetComment() Comment {
	return n.comment
}

func (n *baseNode) SetComment(comment Comment) {
	n.comment = comment
}

func (n *baseNode) Position() Position {
	return n.pos
}

func (n *baseNode) SetPosition(pos Position) {
	n.pos = pos
}

type Document struct {
	baseNode
	Content []Node
}

func (n *Document) Kind() NodeKind {
	return DocumentNode
}

func (n *Document) Clone() Node {
	clone := &Document{
		baseNode: n.baseNode,
		Content:  make([]Node, len(n.Content)),
	}
	for i, node := range n.Content {
		if node != nil {
			clone.Content[i] = node.Clone()
		}
	}
	return clone
}

func (n *Document) String() string {
	return fmt.Sprintf("Document(%d nodes)", len(n.Content))
}

type Scalar struct {
	baseNode
	Value string
	Style ScalarStyle
}

type ScalarStyle int

const (
	PlainStyle ScalarStyle = iota
	SingleQuotedStyle
	DoubleQuotedStyle
	LiteralStyle
	FoldedStyle
	TaggedStyle
)

func (n *Scalar) Kind() NodeKind {
	return ScalarNode
}

func (n *Scalar) Clone() Node {
	return &Scalar{
		baseNode: n.baseNode,
		Value:    n.Value,
		Style:    n.Style,
	}
}

func (n *Scalar) String() string {
	return fmt.Sprintf("Scalar(%s)", n.Value)
}

type Mapping struct {
	baseNode
	Content []*MappingEntry
	Style   CollectionStyle
}

type MappingEntry struct {
	Key     Node
	Value   Node
	Comment Comment
}

type CollectionStyle int

const (
	BlockStyle CollectionStyle = iota
	FlowStyle
)

func (n *Mapping) Kind() NodeKind {
	return MappingNode
}

func (n *Mapping) Clone() Node {
	clone := &Mapping{
		baseNode: n.baseNode,
		Content:  make([]*MappingEntry, len(n.Content)),
		Style:    n.Style,
	}
	for i, entry := range n.Content {
		if entry != nil {
			cloneEntry := &MappingEntry{
				Comment: entry.Comment,
			}
			if entry.Key != nil {
				cloneEntry.Key = entry.Key.Clone()
			}
			if entry.Value != nil {
				cloneEntry.Value = entry.Value.Clone()
			}
			clone.Content[i] = cloneEntry
		}
	}
	return clone
}

func (n *Mapping) String() string {
	return fmt.Sprintf("Mapping(%d entries)", len(n.Content))
}

type SortMode int

const (
	SortAscending SortMode = iota
	SortDescending
	SortOriginal
)

type SortTarget int

const (
	SortKeys SortTarget = iota
	SortValues
	SortBoth
)

func (n *Mapping) Sort(mode SortMode, target SortTarget, compare func(a, b string) int) {
	if target == SortKeys || target == SortBoth {
		n.sortByKeys(mode, compare)
	} else if target == SortValues {
		n.sortByValues(mode, compare)
	}
}

func (n *Mapping) sortByKeys(mode SortMode, compare func(a, b string) int) {
	if compare == nil {
		compare = defaultCompare
	}

	sort.SliceStable(n.Content, func(i, j int) bool {
		keyI := getNodeStringValue(n.Content[i].Key)
		keyJ := getNodeStringValue(n.Content[j].Key)

		result := compare(keyI, keyJ)
		if mode == SortDescending {
			return result > 0
		}
		return result < 0
	})
}

func (n *Mapping) sortByValues(mode SortMode, compare func(a, b string) int) {
	if compare == nil {
		compare = defaultCompare
	}

	sort.SliceStable(n.Content, func(i, j int) bool {
		valI := getNodeStringValue(n.Content[i].Value)
		valJ := getNodeStringValue(n.Content[j].Value)

		result := compare(valI, valJ)
		if mode == SortDescending {
			return result > 0
		}
		return result < 0
	})
}

func defaultCompare(a, b string) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

func getNodeStringValue(node Node) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *Scalar:
		return n.Value
	default:
		return node.String()
	}
}

type Sequence struct {
	baseNode
	Content []Node
	Style   CollectionStyle
}

func (n *Sequence) Kind() NodeKind {
	return SequenceNode
}

func (n *Sequence) Clone() Node {
	clone := &Sequence{
		baseNode: n.baseNode,
		Content:  make([]Node, len(n.Content)),
		Style:    n.Style,
	}
	for i, node := range n.Content {
		if node != nil {
			clone.Content[i] = node.Clone()
		}
	}
	return clone
}

func (n *Sequence) String() string {
	return fmt.Sprintf("Sequence(%d items)", len(n.Content))
}

type Alias struct {
	baseNode
	Identifier string
}

func (n *Alias) Kind() NodeKind {
	return AliasNode
}

func (n *Alias) Clone() Node {
	return &Alias{
		baseNode:   n.baseNode,
		Identifier: n.Identifier,
	}
}

func (n *Alias) String() string {
	return fmt.Sprintf("Alias(%s)", n.Identifier)
}

func NewDocument() *Document {
	return &Document{
		Content: make([]Node, 0),
	}
}

func NewScalar(value string) *Scalar {
	return &Scalar{
		Value: value,
		Style: PlainStyle,
	}
}

func NewMapping() *Mapping {
	return &Mapping{
		Content: make([]*MappingEntry, 0),
		Style:   BlockStyle,
	}
}

func NewSequence() *Sequence {
	return &Sequence{
		Content: make([]Node, 0),
		Style:   BlockStyle,
	}
}

func NewAlias(identifier string) *Alias {
	return &Alias{
		Identifier: identifier,
	}
}
