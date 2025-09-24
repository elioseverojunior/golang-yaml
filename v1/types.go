package yaml

import "golang-yaml/v1/ast"

type Document = ast.Document
type Scalar = ast.Scalar
type Mapping = ast.Mapping
type Sequence = ast.Sequence
type MappingEntry = ast.MappingEntry

const (
	SortAscending  = ast.SortAscending
	SortDescending = ast.SortDescending
	SortOriginal   = ast.SortOriginal
)

const (
	SortKeys   = ast.SortKeys
	SortValues = ast.SortValues
	SortBoth   = ast.SortBoth
)
