# Commons

Commons je sdílený Go modul používaný ostatními backendovými službami RIoT. Soustředí společné datové modely, názvy RabbitMQ front, pomocné typy a utilitní funkce, aby se stejné kontrakty neopakovaly v Backend Core, Message Processing Unit a Time Series Store.

Modul není samostatně spouštěná služba, nemá vlastní kontejner, runtime konfiguraci ani veřejný port. Ostatní Go moduly jej připojují jako lokální závislost přes `replace` v `go.mod` a při Docker buildu se kopíruje společně se zdrojovým modulem.

## Použití

V navazujících modulech se `commons` používá hlavně pro:

- sdílené RabbitMQ konstanty a názvy front
- typy zpráv pro interní servisní komunikaci
- modely KPI definic a stromu KPI podmínek
- modely časových dat, filtrů, kurzorů a čtecích požadavků
- serializaci JSON a CBOR payloadů
- práci s API klíči, hashováním a oprávněními
- pomocné typy jako `Optional`, `Result`, `Pair` a `Set`
- obecné utility pro prostředí, čekání na závislosti, synchronizaci a logování profilovacích údajů

## Struktura modulu

- `go.mod`, `go.sum`: Go modul a závislosti
- `src/rabbitmq/`: klient RabbitMQ, publikování, konzumace zpráv, RPC streamy a dávkové publikování
- `src/sharedConstants/`: sdílené názvy RabbitMQ front a exchange
- `src/sharedModel/apiModel.go`: modely pro API a WebSocket zprávy
- `src/sharedModel/iscMessages.go`: modely zpráv interní servisní komunikace
- `src/sharedModel/kpiDefinitionModel.go`: model KPI definic a uzlů KPI stromu
- `src/sharedModel/messageProcessingUnit.go`: modely runtime stavu používané v MPU
- `src/sharedModel/timeSeries.go`: modely časových dat, filtrů, kurzorů, čtení a reprocessingu
- `src/sharedModel/json.go`: vlastní JSON serializace a deserializace KPI stromů
- `src/sharedUtils/`: obecné pomocné funkce a generické utility

## RabbitMQ knihovna

Balíček `src/rabbitmq/` sjednocuje práci s RabbitMQ napříč backendem. Poskytuje sdílený connection manager, vytváření kanálů, publikování JSON zpráv, konzumaci zpráv s automatickou deserializací, konzumaci z fanout exchange a pomocné funkce pro RPC streamy.

Typické použití:

```go
client := rabbitmq.NewClient()
defer client.Dispose()

err := rabbitmq.ConsumeJSONMessages[sharedModel.KPIFulfillmentCheckRequestTupleISCMessage](
	client,
	sharedConstants.KPIFulfillmentCheckRequestsQueueName,
	func(message sharedModel.KPIFulfillmentCheckRequestTupleISCMessage) error {
		return nil
	},
)
```
