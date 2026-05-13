# MHD preprocesor

MHD preprocesor zpracovává živá data o vozidlech městské hromadné dopravy a převádí je na stavové zprávy pro RIoT. Kombinuje WebSocket stream s GTFS daty, podle jízdních řádů páruje příchozí polohy na konkrétní spoje a vytváří instance typu `MHD_TRIP`. Do systému posílá registrace typu zdroje dat, registrace instancí a průběžné stavy spojů určené pro vyhodnocení KPI.

Modul udržuje runtime stav aktivních spojů. Pokud spoj přestane být aktivní nebo doběhne jeho plánovaný interval, odešle neaktivní stav, aby navazující zpracování dokázalo zaznamenat ukončení jízdy.

## Spuštění

Modul se běžně spouští z kořene repozitáře preprocesorů přes hlavní `docker-compose.yml`:

```bash
docker compose up -d myrta-mhd-preprocessor
```

Preprocesor nemá vlastní veřejný port. Z hlediska kontejnerů potřebuje dostupný RabbitMQ z hlavní sestavy systému a externí zdroje MHD WebSocket a GTFS. Při běhu v samostatné sestavě musí být připojený do stejné Docker sítě jako hlavní RIoT stack, typicky `riot_default`, aby dosáhl na službu `rabbitmq`. Proměnná `RABBITMQ_URL` by proto v kontejneru měla směřovat na `amqp://riot:secret@rabbitmq:5672`, případně na odpovídající hodnotu z hlavní konfigurace.

## Konfigurace

Hlavní proměnné prostředí používané modulem:

- `RABBITMQ_URL`: připojovací řetězec k RabbitMQ
- `MHD_WS_URL`: WebSocket endpoint s živými polohami vozidel
- `MHD_GTFS_URL`: URL GTFS archivu
- `MHD_WS_AUTHORIZATION`: hodnota hlavičky `Authorization` pro WebSocket zdroj
- `MHD_WS_AUTH`: alternativní název proměnné pro WebSocket autorizaci
- `WS_AUTHORIZATION`: obecná fallback proměnná pro WebSocket autorizaci

Pokud `MHD_WS_URL` nebo `MHD_GTFS_URL` nejsou nastavené, modul použije výchozí adresy definované v `src/config.go`.

## RabbitMQ komunikace

Typické přijímané fronty a zprávy:

- `set-of-sd-instances-updates`: aktuální seznam instancí známých Backend Core

Typické odesílané fronty a zprávy:

- `sd-type-registration-requests`: registrace typu zdroje dat `MHD_TRIP`
- `sd-instance-registration-requests`: registrace konkrétních jízd jako SD instancí
- `kpi-fulfillment-check-requests`: stavové zprávy spojů určené pro MPU a vyhodnocení KPI

## Struktura modulu

- `Dockerfile`: build Go aplikace společně s lokálním modulem `commons`
- `go.mod`, `go.sum`: Go modul a závislosti
- `src/main.go`: inicializace modulu, registrace typu, načtení GTFS a spuštění WebSocket loopu
- `src/config.go`: výchozí adresy zdrojů a runtime konfigurace
- `src/gtfs_store.go`: stahování, parsování a ukládání GTFS definic spojů
- `src/ws_source.go`: připojení k WebSocket zdroji a zpracování živých zpráv
- `src/matching.go`: párování živých dat na GTFS spoje
- `src/segment.go`: určení aktivního segmentu mezi zastávkami
- `src/runtime.go`: runtime stav instancí a uzavírání doběhnutých jízd
- `src/publish.go`: publikování registrací a stavových zpráv do RabbitMQ
- `src/sdtype.go`: definice parametrů typu `MHD_TRIP`
- `src/model.go`: datové struktury živých záznamů a GTFS spojů
- `src/helpers.go`: pomocné funkce pro čas, identifikátory, hodnoty a práci s CSV
