# NDIC preprocesor

NDIC preprocesor zpracovává DATEX II snapshoty z NDIC a převádí je na stavové zprávy pro RIoT. Periodicky načítá poslední dostupnou zprávu, parsuje FCD hodnoty pro společnou kategorii `anyVehicle`, doplňuje lokalizační metadata z přibalených TMC tabulek a vytváří instance typu `NDIC_TRAFFIC`.

Modul udržuje runtime stav sledovaných dopravních segmentů. Pokud se dříve aktivní segment v novém snapshotu neobjeví, odešle neaktivní stav, aby navazující zpracování zaznamenalo ukončení nebo zmizení daného záznamu.

## Spuštění

Modul se běžně spouští z kořene repozitáře preprocesorů přes hlavní `docker-compose.yml`:

```bash
docker compose up -d myrta-ndic-preprocessor
```

Preprocesor nemá vlastní veřejný port. Z hlediska kontejnerů potřebuje dostupný RabbitMQ z hlavní sestavy systému a službu poskytující poslední DATEX snapshot, typicky DATEX Downloader. Při běhu v samostatné sestavě musí být připojený do stejné Docker sítě jako hlavní RIoT stack, typicky `riot_default`, aby dosáhl na službu `rabbitmq`. Proměnná `RABBITMQ_URL` by proto v kontejneru měla směřovat na `amqp://riot:secret@rabbitmq:5672`, případně na odpovídající hodnotu z hlavní konfigurace.

## Konfigurace

Hlavní proměnné prostředí používané modulem:

- `RABBITMQ_URL`: připojovací řetězec k RabbitMQ
- `NDIC_URL`: adresa služby poskytující poslední DATEX snapshot, typicky endpoint DATEX Downloaderu

Pokud `NDIC_URL` není nastavené, modul použije výchozí adresu definovanou v `src/config.go`. Pokud adresa neobsahuje konkrétní cestu, doplní se `/api/latest`.

## RabbitMQ komunikace

Typické přijímané fronty a zprávy:

- `set-of-sd-instances-updates`: aktuální seznam instancí známých Backend Core

Typické odesílané fronty a zprávy:

- `sd-type-registration-requests`: registrace typu zdroje dat `NDIC_TRAFFIC`
- `sd-instance-registration-requests`: registrace dopravních segmentů jako SD instancí
- `kpi-fulfillment-check-requests`: stavové zprávy segmentů určené pro MPU a vyhodnocení KPI

## Struktura modulu

- `Dockerfile`: build Go aplikace společně s lokálním modulem `commons` a TMC daty
- `go.mod`, `go.sum`: Go modul a závislosti
- `src/main.go`: inicializace modulu, registrace typu a periodické zpracování NDIC dat
- `src/config.go`: výchozí adresa zdroje, interval zpracování a runtime konfigurace
- `src/fetch.go`: stahování posledního DATEX snapshotu a rozbalení XML z JSON wrapperu
- `src/parser.go`: parsování DATEX II XML a extrakce FCD hodnot
- `src/tmc.go`: načtení TMC tabulek a doplnění lokalizačních metadat
- `src/runtime.go`: runtime stav instancí a uzavírání chybějících segmentů
- `src/publish.go`: publikování registrací a stavových zpráv do RabbitMQ
- `src/sdtype.go`: definice parametrů typu `NDIC_TRAFFIC`
- `src/model.go`: datové struktury parsovaných snapshotů a TMC metadat
- `src/helpers.go`: pomocné funkce pro identifikátory, popisky a práci s časem
- `static_data/tmc/`: přibalená referenční TMC data používaná pro enrichment
