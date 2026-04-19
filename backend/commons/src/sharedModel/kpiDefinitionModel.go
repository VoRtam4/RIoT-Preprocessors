package sharedModel

type LogicalOperationNodeType string

const (
	AND LogicalOperationNodeType = "and"
	OR  LogicalOperationNodeType = "or"
	NOR LogicalOperationNodeType = "nor"
	NOT LogicalOperationNodeType = "not"
)

type KPINodeType string

const (
	StringEQAtom        KPINodeType = "string_eq_atom"
	StringNEQAtom       KPINodeType = "string_neq_atom"
	StringExistsAtom    KPINodeType = "string_exists_atom"
	StringNotExistsAtom KPINodeType = "string_not_exists_atom"

	BooleanEQAtom        KPINodeType = "boolean_eq_atom"
	BooleanNEQAtom       KPINodeType = "boolean_neq_atom"
	BooleanExistsAtom    KPINodeType = "boolean_exists_atom"
	BooleanNotExistsAtom KPINodeType = "boolean_not_exists_atom"

	NumericEQAtom        KPINodeType = "numeric_eq_atom"
	NumericNEQAtom       KPINodeType = "numeric_neq_atom"
	NumericGTAtom        KPINodeType = "numeric_gt_atom"
	NumericGEQAtom       KPINodeType = "numeric_geq_atom"
	NumericLTAtom        KPINodeType = "numeric_lt_atom"
	NumericLEQAtom       KPINodeType = "numeric_leq_atom"
	NumericExistsAtom    KPINodeType = "numeric_exists_atom"
	NumericNotExistsAtom KPINodeType = "numeric_not_exists_atom"

	LogicalOperation KPINodeType = "logical_operation"
)

type SDInstanceMode string

const (
	ALL      SDInstanceMode = "all"
	SELECTED SDInstanceMode = "selected"
)

type KPIDefinition struct {
	ID                    *uint32        `json:"id,omitempty"`
	Label                 string         `json:"label"`
	UserID                *uint32        `json:"userID,omitempty"`
	SDTypeID              uint32         `json:"sdTypeID"`
	SDTypeSpecification   string         `json:"sdTypeSpecification"`
	UserIdentifier        string         `json:"userIdentifier"`
	RootNode              KPINode        `json:"rootNode"`
	SDInstanceMode        SDInstanceMode `json:"sdInstanceMode"`
	SelectedSDInstanceIDs []uint32       `json:"selectedSDInstanceUIDs"`
}

type KPIDefinitionMPU struct {
	ID                     *uint32        `json:"id,omitempty"`
	UserID                 uint32         `json:"userID,omitempty"`
	SDTypeUID              string         `json:"sdTypeSpecification"`
	RootNode               KPINode        `json:"rootNode"`
	SDInstanceMode         SDInstanceMode `json:"sdInstanceMode"`
	SelectedSDInstanceUIDs []string       `json:"selectedSDInstanceUIDs"`
}

type KPINode interface {
	GetType() KPINodeType
}

type StringEQAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
	ReferenceValue           string `json:"referenceValue"`
}

func (*StringEQAtomKPINode) GetType() KPINodeType {
	return StringEQAtom
}

type StringNEQAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
	ReferenceValue           string `json:"referenceValue"`
}

func (*StringNEQAtomKPINode) GetType() KPINodeType {
	return StringNEQAtom
}

type StringExistsAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
}

func (*StringExistsAtomKPINode) GetType() KPINodeType {
	return StringExistsAtom
}

type StringNotExistsAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
}

func (*StringNotExistsAtomKPINode) GetType() KPINodeType {
	return StringNotExistsAtom
}

type BooleanEQAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
	ReferenceValue           bool   `json:"referenceValue"`
}

func (*BooleanEQAtomKPINode) GetType() KPINodeType {
	return BooleanEQAtom
}

type BooleanNEQAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
	ReferenceValue           bool   `json:"referenceValue"`
}

func (*BooleanNEQAtomKPINode) GetType() KPINodeType {
	return BooleanNEQAtom
}

type BooleanExistsAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
}

func (*BooleanExistsAtomKPINode) GetType() KPINodeType {
	return BooleanExistsAtom
}

type BooleanNotExistsAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
}

func (*BooleanNotExistsAtomKPINode) GetType() KPINodeType {
	return BooleanNotExistsAtom
}

type NumericEQAtomKPINode struct {
	SDParameterID            uint32  `json:"sdParameterID"`
	SDParameterSpecification string  `json:"sdParameterSpecification"`
	ReferenceValue           float64 `json:"referenceValue"`
}

func (*NumericEQAtomKPINode) GetType() KPINodeType {
	return NumericEQAtom
}

type NumericNEQAtomKPINode struct {
	SDParameterID            uint32  `json:"sdParameterID"`
	SDParameterSpecification string  `json:"sdParameterSpecification"`
	ReferenceValue           float64 `json:"referenceValue"`
}

func (*NumericNEQAtomKPINode) GetType() KPINodeType {
	return NumericNEQAtom
}

type NumericGTAtomKPINode struct {
	SDParameterID            uint32  `json:"sdParameterID"`
	SDParameterSpecification string  `json:"sdParameterSpecification"`
	ReferenceValue           float64 `json:"referenceValue"`
}

func (*NumericGTAtomKPINode) GetType() KPINodeType {
	return NumericGTAtom
}

type NumericGEQAtomKPINode struct {
	SDParameterID            uint32  `json:"sdParameterID"`
	SDParameterSpecification string  `json:"sdParameterSpecification"`
	ReferenceValue           float64 `json:"referenceValue"`
}

func (*NumericGEQAtomKPINode) GetType() KPINodeType {
	return NumericGEQAtom
}

type NumericLTAtomKPINode struct {
	SDParameterID            uint32  `json:"sdParameterID"`
	SDParameterSpecification string  `json:"sdParameterSpecification"`
	ReferenceValue           float64 `json:"referenceValue"`
}

func (*NumericLTAtomKPINode) GetType() KPINodeType {
	return NumericLTAtom
}

type NumericLEQAtomKPINode struct {
	SDParameterID            uint32  `json:"sdParameterID"`
	SDParameterSpecification string  `json:"sdParameterSpecification"`
	ReferenceValue           float64 `json:"referenceValue"`
}

func (*NumericLEQAtomKPINode) GetType() KPINodeType {
	return NumericLEQAtom
}

type NumericExistsAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
}

func (*NumericExistsAtomKPINode) GetType() KPINodeType {
	return NumericExistsAtom
}

type NumericNotExistsAtomKPINode struct {
	SDParameterID            uint32 `json:"sdParameterID"`
	SDParameterSpecification string `json:"sdParameterSpecification"`
}

func (*NumericNotExistsAtomKPINode) GetType() KPINodeType {
	return NumericNotExistsAtom
}

type LogicalOperationKPINode struct {
	Type       LogicalOperationNodeType `json:"type"`
	ChildNodes []KPINode                `json:"childNodes"`
}

func (*LogicalOperationKPINode) GetType() KPINodeType {
	return LogicalOperation
}
