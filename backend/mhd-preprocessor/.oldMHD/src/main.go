/**
 * @File: main.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: Preprocessor přijímá, zpracovává a posílá dále do systému real-time data o poloze vozidel MHD Brno.
 */

package main

import (
    "github.com/gorilla/websocket"
    "github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
    "github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
    "github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
    "github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
    "log"
    "fmt"
    "time"
    "os"
    "sync"
    "io"
    "net/http"
)

const WS_URL = "wss://gis.brno.cz/geoevent/ws/services/ODAE_public_transit_stream_service/StreamServer/subscribe"
const rabbitMQURL = "amqp://guest:guest@172.25.0.6:5672/"

var (
    sdTypes          = sharedUtils.NewSet[string]()
    sdTypesMutex     sync.Mutex
    sdInstances      = sharedUtils.NewSet[sharedModel.SDInstanceInfo]()
    sdInstancesMutex sync.Mutex
)

// Připojení k RabbitMQ.
func connectToRabbitMQ() rabbitmq.Client {
    client := rabbitmq.NewClient()
    log.Println("Successfully connected to RabbitMQ.")
    return client
}

// Získání aktualizovaných SD typů.
func checkForSetOfSDTypesUpdates(client rabbitmq.Client) {
    err := rabbitmq.ConsumeJSONMessages[sharedModel.SDTypeConfigurationUpdateISCMessage](
        client, sharedConstants.SetOfSDTypesUpdatesQueueName,
        func(messagePayload sharedModel.SDTypeConfigurationUpdateISCMessage) error {
            updatedSDTypes := sharedUtils.NewSetFromSlice(messagePayload)
            sdTypesMutex.Lock()
            sdTypes = updatedSDTypes
            sdTypesMutex.Unlock()
            return nil
        })
    if err != nil {
        log.Printf("Failed to consume messages from '%s'", sharedConstants.SetOfSDTypesUpdatesQueueName)
    }
}

// Získání aktualizovaných SD instancí.
func checkForSetOfSDInstancesUpdates(client rabbitmq.Client) {
    err := rabbitmq.ConsumeJSONMessages[sharedModel.SDInstanceConfigurationUpdateISCMessage](
        client, sharedConstants.SetOfSDInstancesUpdatesQueueName,
        func(messagePayload sharedModel.SDInstanceConfigurationUpdateISCMessage) error {
            updatedSDInstances := sharedUtils.NewSetFromSlice(messagePayload)
            sdInstancesMutex.Lock()
            sdInstances = updatedSDInstances
            sdInstancesMutex.Unlock()
            return nil
        })
    if err != nil {
        log.Printf("Failed to consume messages from '%s'", sharedConstants.SetOfSDInstancesUpdatesQueueName)
    }
}

// Rozpoznání scénáře SD instance
func determineSDInstanceScenario(uid string) string {
    sdInstancesMutex.Lock()
    defer sdInstancesMutex.Unlock()

    if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: true}) {
        return "confirmed"
    } else if sdInstances.Contains(sharedModel.SDInstanceInfo{SDInstanceUID: uid, ConfirmedByUser: false}) {
        return "notYetConfirmed"
    }
    return "unknown"
}

// Připojení k WebSocketu a čtení zpráv.
func fetchWSData(client rabbitmq.Client) {
    log.Println("Starting WebSocket connection...")
    conn, _, err := websocket.DefaultDialer.Dial(WS_URL, nil)
    if err != nil {
        log.Fatalf("Failed to connect to WebSocket: %v", err)
    }
    log.Println("WebSocket connection established, listening for messages...")
    defer conn.Close()

    log.Println("Waiting for a message from WebSocket...")
    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Printf("Error reading WebSocket message: %v", err)
            break
        }
        processWebSocketMessage(message, client)
    }
}

// Vrátí hodnotu typu float z mapy atributů podle zadaného klíče.
func extractFloatAttribute(attributes map[string]interface{}, key string) float64 {
    if val, ok := attributes[key]; ok {
        switch v := val.(type) {
        case float64:
            return v
        case float32:
            return float64(v)
        }
    }
    return 0
}

// Získá stabilní hash pro kombinaci lineID a routeID přes volání GTFS backendu.
func resolveStableTripHash(lineID, routeID string) (string, error) {
    url := fmt.Sprintf("http://gtfs-backend:9100/gtfs/resolve-trip-hash?lineid=%s&routeid=%s", lineID, routeID)
    resp, err := http.Get(url)
    if err != nil {
        return "", fmt.Errorf("HTTP error: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("GTFS-core returned status: %s", resp.Status)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("read error: %v", err)
    }

    return string(body), nil
}

// Zpracování zpráv z WebSocketu.
func processWebSocketMessage(message []byte, client rabbitmq.Client) {
    result := sharedUtils.DeserializeFromJSON[map[string]interface{}](message)
    if result.IsFailure() {
        log.Printf("Error deserializing message: %v", result.GetError())
        return
    }

    data := result.GetPayload()
    attributes, ok := data["attributes"].(map[string]interface{})
    if !ok {
        log.Println("Missing 'attributes' in message")
        return
    }

    lineID := extractStringAttribute(attributes, "lineid")
    routeID := extractStringAttribute(attributes, "routeid")
    stableHash, err := resolveStableTripHash(lineID, routeID)
    if err != nil {
        log.Printf("Trip hash not found for lineID=%s and routeID=%s: %v", lineID, routeID, err)
        return
    }

    // Přidání všech dalších atributů.
    parameters := make(map[string]interface{})
    for k, v := range attributes {
        parameters[k] = v
    }

    instanceUID := fmt.Sprintf("MHD_TRIP_%s", stableHash)
    sdType := "MHD"
    scenario := determineSDInstanceScenario(instanceUID)

    generateKPIRequest(instanceUID, sdType, parameters, client)

    if scenario == "unknown" {
        generateSDInstanceRegistrationRequest(instanceUID, sdType, float32(time.Now().Unix()), client)
    }
}

// Vrátí hodnotu typu string z mapy atributů podle zadaného klíče.
func extractStringAttribute(attributes map[string]interface{}, key string) string {
    raw, ok := attributes[key]
    if !ok {
        return ""
    }
    return fmt.Sprintf("%v", raw)
}

// Vytvoří a odešle KPI požadavek do fronty RabbitMQ.
func generateKPIRequest(uid, sdType string, parameters map[string]interface{}, client rabbitmq.Client) {
    message := sharedModel.KPIFulfillmentCheckRequestISCMessage{
        Timestamp:          float32(time.Now().Unix()),
        SDInstanceUID:       uid,
        SDTypeSpecification: sdType,
        Parameters:          parameters,
    }

    jsonMessage := sharedUtils.SerializeToJSON(message)
    if jsonMessage.IsFailure() {
        log.Printf("Error serializing KPI request: %v", jsonMessage.GetError())
        return
    }

    err := client.PublishJSONMessage(
        sharedUtils.NewEmptyOptional[string](),
        sharedUtils.NewOptionalOf(sharedConstants.KPIFulfillmentCheckRequestsQueueName),
        jsonMessage.GetPayload(),
    )

    if err != nil {
        log.Printf("Error publishing KPI request: %v", err)
    } else {
        log.Println("KPI request successfully published")
    }
}

// Vytvoří a odešle žádost o registraci SD instance do RabbitMQ a uloží ji do seznamu SD instancí.
func generateSDInstanceRegistrationRequest(uid string, sdType string, timestamp float32, client rabbitmq.Client) {
    log.Printf("[DEBUG] Preparing to publish SDInstanceRegistrationRequest: UID=%s, Type=%s, Timestamp=%f", uid, sdType, timestamp)

    message := sharedModel.SDInstanceRegistrationRequestISCMessage{
        Timestamp:           timestamp,
        SDInstanceUID:       uid,
        SDTypeSpecification: sdType,
    }

    jsonMessage := sharedUtils.SerializeToJSON(message)
    if jsonMessage.IsFailure() {
        log.Printf("[ERROR] Failed to serialize SDInstanceRegistrationRequest: %v", jsonMessage.GetError())
        return
    }

    err := client.PublishJSONMessage(
        sharedUtils.NewEmptyOptional[string](),
        sharedUtils.NewOptionalOf(sharedConstants.SDInstanceRegistrationRequestsQueueName),
        jsonMessage.GetPayload(),
    )
    if err != nil {
        log.Printf("[ERROR] Failed to publish SDInstanceRegistrationRequest: %v", err)
    } else {
        log.Printf("[DEBUG] Successfully published SDInstanceRegistrationRequest to queue %s", sharedConstants.SDInstanceRegistrationRequestsQueueName)
    }

    sdInstancesMutex.Lock()
    sdInstances.Add(sharedModel.SDInstanceInfo{
        SDInstanceUID:   uid,
        ConfirmedByUser: false,
    })
    sdInstancesMutex.Unlock()
}

func main() {
    log.SetOutput(os.Stderr)

    client := connectToRabbitMQ()
    defer client.Dispose()

    go checkForSetOfSDTypesUpdates(client)
    go checkForSetOfSDInstancesUpdates(client)

    fetchWSData(client)
}
