/**
 * @file apiModel.go
 * @brief Sdílené API modely potřebné pro registrační a datové zprávy preprocesorů.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedModel

type MessageType string

const (
	MessageRequest     MessageType = "request"
	MessageResponse    MessageType = "response"
	MessageEvent       MessageType = "event"
	MessageSubscribe   MessageType = "subscribe"
	MessageUnsubscribe MessageType = "unsubscribe"
	MessageError       MessageType = "error"
)

type WebSocketMessage struct {
	Type    MessageType `json:"type"`
	ID      string      `json:"id,omitempty"`
	Action  string      `json:"action,omitempty"`
	Topic   string      `json:"topic,omitempty"`
	Payload any         `json:"payload,omitempty"`
	Success bool        `json:"success,omitempty"`
	Error   string      `json:"error,omitempty"`
}
