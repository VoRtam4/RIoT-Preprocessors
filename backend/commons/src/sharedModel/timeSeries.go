package sharedModel

import (
	"time"
)

type TimeSeriesType string

const (
	TimeSeriesTypeRaw       TimeSeriesType = "raw"
	TimeSeriesTypeKPIResult TimeSeriesType = "kpi"
)

type ParameterRole string

const (
	ParameterRoleBase  ParameterRole = "base"
	ParameterRoleTime  ParameterRole = "time"
	ParameterRoleTag   ParameterRole = "tag"
	ParameterRoleField ParameterRole = "field"
)

type FilterNodeType string

const (
	FilterNodeTypeLogical FilterNodeType = "logical"
	FilterNodeTypeRule    FilterNodeType = "rule"
)

type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "and"
	LogicalOr  LogicalOperator = "or"
	LogicalNot LogicalOperator = "not"
)

type FilterOperator string

const (
	OpEQ       FilterOperator = "eq"
	OpNEQ      FilterOperator = "neq"
	OpContains FilterOperator = "contains"
	OpIn       FilterOperator = "in"
	OpPrefix   FilterOperator = "prefix"
	OpSuffix   FilterOperator = "suffix"
	OpRegex    FilterOperator = "regex"
)

type TimeSeriesRawRecord struct {
	EventTime     time.Time              `json:"eventTime"`
	SDInstanceUID string                 `json:"sdInstanceUID"`
	SDTypeUID     string                 `json:"sdTypeID"`
	Fields        map[string]interface{} `json:"fields"`
	Tags          map[string]string      `json:"tags"`
}

type TimeSeriesKPIResultRecord struct {
	JobID           string            `json:"jobId"`
	EventTime       time.Time         `json:"eventTime"`
	SDInstanceUID   string            `json:"sdInstanceUID"`
	SDTypeUID       string            `json:"sdTypeID"`
	KPIDefinitionID uint32            `json:"kpiDefinitionID"`
	Fulfilled       bool              `json:"fulfilled"`
	Tags            map[string]string `json:"tags"`
}

type TimeSeriesReprocessRequest struct {
	SDTypeUID      string    `json:"SDTypeID"`
	SDInstanceUIDs []string  `json:"sdInstanceUID,omitempty"`
	To             time.Time `json:"to"`
	Batch          int       `json:"batch"`
}

type TimeSeriesReprocessReadRequest struct {
	JobID           string    `json:"jobId"`
	Wait            bool      `json:"wait"`
	KPIDefinitionID uint32    `json:"kpiDefinitionID"`
	SDTypeUID       string    `json:"SDTypeID"`
	SDInstanceUIDs  []string  `json:"sdInstanceUIDs,omitempty"`
	To              time.Time `json:"to"`
	Batch           int       `json:"batch"`
}

type TimeSeriesReprocessReadResponse struct {
	Data    []TimeSeriesDataPoint `json:"data"`
	HasMore bool                  `json:"hasMore"`
	Error   string                `json:"error,omitempty"`
}

type TimeSeriesDataPoint struct {
	Time time.Time              `json:"time"`
	Tags map[string]string      `json:"tags,omitempty"`
	Data map[string]interface{} `json:"data"`
}

type TimeSeriesReadRequest struct {
	Type             TimeSeriesType    `json:"type"`
	SDTypeUID        string            `json:"SDTypeID,omitempty"`
	SDInstanceUIDs   []string          `json:"sdInstanceUIDs,omitempty"`
	KPIDefinitionIDs []uint32          `json:"kpiDefinitionIDs,omitempty"`
	From             *time.Time        `json:"from,omitempty"`
	To               *time.Time        `json:"to,omitempty"`
	AggregateSeconds *int              `json:"aggregateSeconds,omitempty"`
	Limit            *int              `json:"limit,omitempty"`
	SortDesc         *bool             `json:"sortDesc,omitempty"`
	Batch            *int              `json:"batch,omitempty"`
	Filters          *FilterNode       `json:"filters,omitempty"`
	Cursor           *TimeSeriesCursor `json:"cursor,omitempty"`
}

type TimeSeriesDistinctTagValuesRequest struct {
	Type             TimeSeriesType `json:"type"`
	SDTypeUID        string         `json:"SDTypeID,omitempty"`
	SDInstanceUIDs   []string       `json:"sdInstanceUIDs,omitempty"`
	KPIDefinitionIDs []uint32       `json:"kpiDefinitionIDs,omitempty"`
	From             *time.Time     `json:"from,omitempty"`
	To               *time.Time     `json:"to,omitempty"`
	Tag              string         `json:"tag"`
	Filters          *FilterNode    `json:"filters,omitempty"`
}

type TimeSeriesDistinctTagValuesResponse struct {
	Values []string `json:"values,omitempty"`
	Error  string   `json:"error,omitempty"`
}

type TimeSeriesReadResponse struct {
	//Parameters     []TimeSeriesParameter            `json:"parameters,omitempty"`
	Base           map[string]string     `json:"base,omitempty"`
	Data           []TimeSeriesDataPoint `json:"data,omitempty"`
	HasMoreBatches bool                  `json:"hasMoreBatches,omitempty"`
	HasMoreData    bool                  `json:"hasMoreData,omitempty"`
	NextCursor     *TimeSeriesCursor     `json:"nextCursor,omitempty"`
	Error          string                `json:"error,omitempty"`
}

type TimeSeriesParameter struct { // Tohle Je potřeba, ale získám z Relační DB, jak vypadá SDType, to v TS DB není
	Denotation string        `json:"denotation,omitempty"`
	Label      string        `json:"label,omitempty"`
	Role       ParameterRole `json:"role,omitempty"`
}

type TimeSeriesCursor struct {
	Time            time.Time `json:"time"`
	SDInstanceUID   string    `json:"sdInstanceUID"`
	KPIDefinitionID *uint32   `json:"kpiDefinitionID,omitempty"`
}

type FilterRule struct {
	Tag      string         `json:"tag"`
	Operator FilterOperator `json:"operator"`
	Value    string         `json:"value"`
}

type FilterNode struct {
	Type     FilterNodeType  `json:"type"`
	Operator LogicalOperator `json:"operator,omitempty"`
	Nodes    []FilterNode    `json:"nodes,omitempty"`
	Rule     *FilterRule     `json:"rule,omitempty"`
}

type QueryPlan struct {
	Type TimeSeriesType

	From time.Time
	To   time.Time

	SDTypeUID        string
	SDInstanceUIDs   []string
	KPIDefinitionIDs []uint32

	Limit    int
	Batch    int
	SortDesc bool
	UseSort  bool

	AggregateSeconds *int

	Cursor    *TimeSeriesCursor
	UseCursor bool

	Measurement string

	UseAggregation    bool
	NeedInitial       bool
	NeedGrouping      bool
	HasInstanceFilter bool
	HasTagFilter      bool
	HasKPI            bool
	IsKPI             bool

	MeasurementFilterFlux string
	InstanceFilterFlux    string
	KPIFilterFlux         string
	TagFilterFlux         string
}
