package sharedModel

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

func marshalKPINode(kpiNode KPINode) ([]byte, error) {
	nodeValue := reflect.Indirect(reflect.ValueOf(kpiNode))
	numberOfFields := nodeValue.NumField()
	nodeMap := make(map[string]any, numberOfFields+1)
	for index := 0; index < numberOfFields; index++ {
		jsonTag := strings.Split(nodeValue.Type().Field(index).Tag.Get("json"), ",")[0]
		nodeMap[jsonTag] = nodeValue.Field(index).Interface()
	}
	nodeMap["kpiNodeType"] = kpiNode.GetType()
	return json.Marshal(nodeMap)
}

func (s *StringEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(s)
}

func (n *StringNEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *StringExistsAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *StringNotExistsAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (b *BooleanEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(b)
}

func (n *BooleanNEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *BooleanExistsAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *BooleanNotExistsAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericNEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericGTAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericGEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericLTAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericLEQAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericExistsAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (n *NumericNotExistsAtomKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(n)
}

func (l *LogicalOperationKPINode) MarshalJSON() ([]byte, error) {
	return marshalKPINode(l)
}

func getKPINode(kpiNodeType KPINodeType) KPINode {
	switch kpiNodeType {
	case StringEQAtom:
		return new(StringEQAtomKPINode)
	case StringNEQAtom:
		return new(StringNEQAtomKPINode)
	case StringExistsAtom:
		return new(StringExistsAtomKPINode)
	case StringNotExistsAtom:
		return new(StringNotExistsAtomKPINode)
	case BooleanEQAtom:
		return new(BooleanEQAtomKPINode)
	case BooleanNEQAtom:
		return new(BooleanNEQAtomKPINode)
	case BooleanExistsAtom:
		return new(BooleanExistsAtomKPINode)
	case BooleanNotExistsAtom:
		return new(BooleanNotExistsAtomKPINode)
	case NumericEQAtom:
		return new(NumericEQAtomKPINode)
	case NumericNEQAtom:
		return new(NumericNEQAtomKPINode)
	case NumericGTAtom:
		return new(NumericGTAtomKPINode)
	case NumericGEQAtom:
		return new(NumericGEQAtomKPINode)
	case NumericLTAtom:
		return new(NumericLTAtomKPINode)
	case NumericLEQAtom:
		return new(NumericLEQAtomKPINode)
	case NumericExistsAtom:
		return new(NumericExistsAtomKPINode)
	case NumericNotExistsAtom:
		return new(NumericNotExistsAtomKPINode)
	case LogicalOperation:
		return new(LogicalOperationKPINode)
	}
	panic(fmt.Errorf("unknown kpi node type: %s", kpiNodeType))
}

func unmarshalKPINode(data []byte) (KPINode, error) {
	kpiNodeTypeJSONDeserializationResult := sharedUtils.DeserializeFromJSON[struct {
		KPINodeType KPINodeType `json:"kpiNodeType"`
	}](data)
	if kpiNodeTypeJSONDeserializationResult.IsFailure() {
		return nil, kpiNodeTypeJSONDeserializationResult.GetError()
	}
	kpiNode := getKPINode(kpiNodeTypeJSONDeserializationResult.GetPayload().KPINodeType)
	err := json.Unmarshal(data, kpiNode)
	return kpiNode, err
}

func (l *LogicalOperationKPINode) UnmarshalJSON(data []byte) error {
	type LogicalOperationKPINodeTA LogicalOperationKPINode
	aux := &struct {
		*LogicalOperationKPINodeTA
		ChildNodes []json.RawMessage `json:"childNodes"`
	}{
		LogicalOperationKPINodeTA: (*LogicalOperationKPINodeTA)(l),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	for _, rawKPINode := range aux.ChildNodes {
		kpiNode, err := unmarshalKPINode(rawKPINode)
		if err != nil {
			return err
		}
		l.ChildNodes = append(l.ChildNodes, kpiNode)
	}
	return nil
}

func (k *KPIDefinitionMPU) UnmarshalJSON(data []byte) error {
	type KPIDefinitionMPUTA KPIDefinitionMPU
	aux := &struct {
		RootNode json.RawMessage `json:"rootNode"`
		*KPIDefinitionMPUTA
	}{
		KPIDefinitionMPUTA: (*KPIDefinitionMPUTA)(k),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	rootKPINode, err := unmarshalKPINode(aux.RootNode)
	if err != nil {
		return err
	}
	k.RootNode = rootKPINode
	return nil
}

func (k KPIDefinition) MarshalJSON() ([]byte, error) {
	type Alias KPIDefinition
	rootNodeBytes, err := marshalKPINode(k.RootNode)
	if err != nil {
		return nil, err
	}
	aux := struct {
		Alias
		RootNode json.RawMessage `json:"rootNode"`
	}{
		Alias:    Alias(k),
		RootNode: rootNodeBytes,
	}
	return json.Marshal(aux)
}

func (k KPIDefinitionMPU) MarshalJSON() ([]byte, error) {
	type Alias KPIDefinitionMPU
	rootNodeBytes, err := marshalKPINode(k.RootNode)
	if err != nil {
		return nil, err
	}
	aux := struct {
		Alias
		RootNode json.RawMessage `json:"rootNode"`
	}{
		Alias:    Alias(k),
		RootNode: rootNodeBytes,
	}
	return json.Marshal(aux)
}
