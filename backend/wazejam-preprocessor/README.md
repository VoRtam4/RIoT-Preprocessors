# Waze Jam preprocesor

Waze Jam preprocesor zpracovává dopravní zácpy z Waze Partner feedu a převádí je na stavové zprávy pro RIoT. Periodicky stahuje JSON feed, z jednotlivých záznamů vytváří agregace podle silničních segmentů a zakládá instance typu `WAZE_JAM_LOCATION`.

Modul udržuje runtime stav aktivních segmentů. Pokud segment z feedu zmizí, odešle nulový stav, aby navazující zpracování zaznamenalo ukončení zácpy nebo dopravního omezení.

## Spuštění

Modul se běžně spouští z kořene repozitáře preprocesorů přes hlavní `docker-compose.yml`:

```bash
docker compose up -d myrta-wazejam-preprocessor
```

Preprocesor nemá vlastní veřejný port. Z hlediska kontejnerů potřebuje dostupný RabbitMQ z hlavní sestavy systému a externí Waze Partner feed. Při běhu v samostatné sestavě musí být připojený do stejné Docker sítě jako hlavní RIoT stack, typicky `riot_default`, aby dosáhl na službu `rabbitmq`. Proměnná `RABBITMQ_URL` by proto v kontejneru měla směřovat na `amqp://riot:secret@rabbitmq:5672`, případně na odpovídající hodnotu z hlavní konfigurace.

## Konfigurace

Hlavní proměnné prostředí používané modulem:

- `RABBITMQ_URL`: připojovací řetězec k RabbitMQ
- `WAZE_URL`: adresa Waze Partner feedu ve formátu JSON

Pokud `WAZE_URL` není nastavené, modul použije výchozí adresu definovanou v `src/config.go`.

## RabbitMQ komunikace

Typické přijímané fronty a zprávy:

- `set-of-sd-instances-updates`: aktuální seznam instancí známých Backend Core

Typické odesílané fronty a zprávy:

- `sd-type-registration-requests`: registrace typu zdroje dat `WAZE_JAM_LOCATION`
- `sd-instance-registration-requests`: registrace silničních segmentů jako SD instancí
- `kpi-fulfillment-check-requests`: stavové zprávy segmentů určené pro MPU a vyhodnocení KPI

## Struktura modulu

- `Dockerfile`: build Go aplikace společně s lokálním modulem `commons`
- `go.mod`, `go.sum`: Go modul a závislosti
- `src/main.go`: inicializace modulu, registrace typu a periodické zpracování Waze feedu
- `src/config.go`: výchozí adresa Waze feedu a runtime konfigurace
- `src/lifecycle.go`: načítání feedu, správa známých instancí a životní cyklus aktivních segmentů
- `src/aggregation.go`: agregace záznamů podle segmentu a směru
- `src/parsing.go`: normalizace hodnot, segmentů a identifikátorů
- `src/publish.go`: publikování registrací a stavových zpráv do RabbitMQ
