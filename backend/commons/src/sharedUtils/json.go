/**
 * @file json.go
 * @brief Pomocné funkce pro JSON serializaci, deserializaci a porovnávání.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

import (
	"encoding/json"
)

func DeserializeFromJSON[T any](data []byte) Result[T] {
	var object T
	err := json.Unmarshal(data, &object)
	if err != nil {
		return NewFailureResult[T](err)
	}
	return NewSuccessResult[T](object)
}

func SerializeToJSON(object any) Result[[]byte] {
	data, err := json.Marshal(object)
	if err != nil {
		return NewFailureResult[[]byte](err)
	}
	return NewSuccessResult[[]byte](data)
}

func CompareJSONs(a interface{}, b interface{}) bool {
	aj := SerializeToJSON(a)
	bj := SerializeToJSON(b)
	if aj.IsFailure() || bj.IsFailure() {
		return false
	}
	return string(aj.GetPayload()) == string(bj.GetPayload())
}
