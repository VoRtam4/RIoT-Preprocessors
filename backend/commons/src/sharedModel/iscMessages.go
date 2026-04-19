package sharedModel

import "time"

type RawDataPointISCMessageTupleISCMessage []RawDataPointISCMessage
type RawDataPointISCMessage struct {
	SDTypeUID     string    `json:"sdTypeUID"`
	SDInstanceUID string    `json:"sdInstanceUID"`
	EventTime     time.Time `json:"eventTime"`
	Payload       []byte    `json:"payload"`
}

type KPIFulfillmentCheckResultTupleISCMessage struct {
	Tuple     []KPIFulfillmentCheckResultISCMessage `json:"sdTypeUID"`
	Reprocess bool                                  `json:"reprocess"`
}

type KPIFulfillmentCheckResultISCMessage struct {
	SDTypeUID       string    `json:"sdTypeUID"`
	SDInstanceUID   string    `json:"sdInstanceUID"`
	KPIDefinitionID uint32    `json:"kpiDefinitionID"`
	EventTime       time.Time `json:"eventTime"`
	Fulfilled       bool      `json:"fulfilled"`
}

type KPIFulfillmentCheckRequestISCMessage struct {
	EventTime     time.Time `json:"eventTime"`
	SDInstanceUID string    `json:"sdInstanceUID"`
	SDTypeUID     string    `json:"sdTypeUID"`
	Parameters    any       `json:"parameters"`
}

type SDInstanceRegistrationRequestISCMessage struct {
	EventTime     time.Time `json:"eventTime"`
	Label         string    `json:"label"`
	SDInstanceUID string    `json:"sdInstanceUID"`
	SDTypeUID     string    `json:"sdTypeUID"`
}

type SDInstanceInfo struct {
	SDInstanceUID   string `json:"sdInstanceUID"`
	ConfirmedByUser bool   `json:"confirmedByUser"`
}

type SDInstanceConfigurationUpdateISCMessage []SDInstanceInfo

type KPIConfigurationUpdateISCMessage struct {
	KpiConfiguration map[string][]KPIDefinitionMPU `json:"kpiConfiguration"`
	JobID            string                        `json:"jobID"`
}

type MessageProcessingUnitConnectionNotification struct{}

type SDTypeConfigurationUpdateISCMessage []SDTypeRegistrationRequestISCMessage
type SDTypeRegistrationRequestISCMessage struct {
	SDTypeUID  string        `json:"sdTypeUID"`
	Label      string        `json:"label"`
	Parameters []SDParameter `json:"parameters"`
}

type SDTypeUpdateTupleISCMessage []SDTypeUpdateISCMessage

type SDTypeUpdateISCMessage struct {
	SDTypeUID  string        `json:"sdTypeUID"`
	Label      string        `json:"label"`
	Parameters []SDParameter `json:"parameters"`
}

type SDParameterType string

const (
	SDParameterTypeString  SDParameterType = "string"
	SDParameterTypeBoolean SDParameterType = "boolean"
	SDParameterTypeNumber  SDParameterType = "number"
)

type SDParameterRole string

const (
	SDParameterRoleField SDParameterRole = "field"
	SDParameterRoleTag   SDParameterRole = "tag"
)

type SDParameter struct {
	Denotation string          `json:"denotation"`
	Type       SDParameterType `json:"type"`
	Label      string          `json:"label"`
	Role       SDParameterRole `json:"role"`
}

type KPIReprocessRequestISCMessage struct {
	JobID           string    `json:"jobId"`
	Wait            bool      `json:"wait"`
	KPIDefinitionID uint32    `json:"kpiDefinitionID"`
	SDTypeUID       string    `json:"sdTypeUID"`
	SDInstanceUIDs  []string  `json:"sdInstanceUIDs,omitempty"`
	To              time.Time `json:"to"`
}

type KPIDeleteResultsRequestISCMessage struct {
	JobID           string `json:"jobId"`
	SDTypeUID       string `json:"sdTypeID"`
	KPIDefinitionID uint32 `json:"kpiDefinitionID"`
}
