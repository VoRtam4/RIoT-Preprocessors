package sharedUtils

import (
	"encoding/base64"

	"github.com/fxamacker/cbor/v2"
)

func EncodeCBOR(m map[string]interface{}) (string, error) {
	b, err := cbor.Marshal(m)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}

func DecodeCBOR(s string) (map[string]interface{}, error) {
	data, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = cbor.Unmarshal(data, &m)
	return m, err
}
