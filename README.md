# RIoT Preprocesory

[![RIoT Platform: organization](https://img.shields.io/badge/RIoT_Platform-organization-blue?logo=github)](https://github.com/RIoT-Platform)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

RIoT Preprocesory obsahují integrační služby, které připojují konkrétní dopravní zdroje k platformě RIoT. Preprocesory načítají data z externích zdrojů, převádějí je do jednotného interního formátu RIoT a přes RabbitMQ publikují registrace typů zdrojů dat, registrace instancí a průběžné stavové zprávy určené pro vyhodnocení KPI.

Tento repozitář vznikl v rámci bakalářské práce **Vojtěcha Hubáčka** jako doplněk k jádru systému RIoT. Práce navazuje na původní systém RIoT **Michala Bureše** a na řešení RTAlerts **Dominika Vondrušky**, ze kterého vychází doména otevřených dopravních dat. Zatímco hlavní repozitář RIoT obsahuje obecné serverové a klientské jádro, tento repozitář řeší připojení reálných zdrojů: Waze Partner feedu, NDIC dat ve formátu DATEX II a živých dat IDS JMK/MHD kombinovaných s GTFS.

![RIoT schema](docs/riot-analytics-scheme.svg)

## Moduly

- [Waze Jam preprocesor](backend/wazejam-preprocessor/) ([README](backend/wazejam-preprocessor/README.md)) - periodicky načítá Waze Partner feed, agreguje zácpy podle silničních segmentů a publikuje stavové zprávy typu `WAZE_JAM_LOCATION`.
- [NDIC preprocesor](backend/ndic-preprocessor/) ([README](backend/ndic-preprocessor/README.md)) - zpracovává DATEX II snapshoty z NDIC, doplňuje TMC metadata a publikuje stavy dopravních segmentů typu `NDIC_TRAFFIC`.
- [MHD preprocesor](backend/mhd-preprocessor/) ([README](backend/mhd-preprocessor/README.md)) - kombinuje WebSocket stream živých poloh vozidel s GTFS daty, páruje vozidla na spoje a publikuje stavy typu `MHD_TRIP`.
- [Commons](backend/commons/) ([README](backend/commons/README.md)) - sdílený Go modul s RabbitMQ kontrakty, datovými modely a utilitami používanými preprocesory.
- [DATEX Downloader](external/datex-downloader/) ([README](external/datex-downloader/README.md)) - pomocná FastAPI služba pro příjem NDIC DATEX II push zpráv a zpřístupnění posledního snapshotu NDIC preprocesoru. Běžně se lokálně nespouští, pokud NDIC preprocesor používá hostovanou instanci této služby.

## Spuštění

Pro běžné spuštění stačí Docker, Docker Compose, hlavní `Makefile` a dostupný repozitář RIoT. Výchozí cesta k hlavnímu repozitáři je `../RIoT`. Změnit ji lze přes proměnnou `RIOT_DIR`.

```bash
make build
```

Příkaz sestaví a spustí hlavní RIoT stack a následně sestaví a spustí preprocesory. Pokud už jsou image sestavené, lze použít:

```bash
make run
```

Zastavení obou stacků:

```bash
make stop
```

Pokud je hlavní repozitář RIoT v jiné složce:

```bash
make build RIOT_DIR=/cesta/k/RIoT
```

Další cíle, například logy, stav kontejnerů nebo vyčištění lokálních volume, vypíše:

```bash
make help
```

Výchozí konfigurace preprocesorů je v souboru [.env](.env). Hlavní RIoT stack používá vlastní konfiguraci ve svém repozitáři.

## Napojení Na RIoT

Preprocesory běží jako samostatný Docker Compose stack, ale musí být připojené do stejné Docker sítě jako hlavní RIoT stack. Výchozí konfigurace používá externí síť `riot_default`, kterou vytvoří hlavní RIoT compose.

Z pohledu preprocesorů musí být RabbitMQ dostupné pod názvem služby `rabbitmq` a proměnná `RABBITMQ_URL` typicky odpovídá:

```text
amqp://riot:secret@rabbitmq:5672
```

Preprocesory používají sdílený modul [Commons](backend/commons/README.md), aby názvy front a datové modely odpovídaly kontraktům hlavního systému. Publikovaná data dále zpracovává hlavně Message Processing Unit v RIoT jádru.

## Porty

Preprocesory MHD, NDIC a Waze nemají vlastní veřejné HTTP porty. Komunikují odchozími požadavky na externí zdroje a přes RabbitMQ do hlavního RIoT stacku.

| Služba | Adresa | Poznámka |
| --- | --- | --- |
| DATEX Downloader | <http://localhost:8000> | pouze při lokálním spuštění `datex-server` |
| RabbitMQ AMQP | `rabbitmq:5672` | dostupné uvnitř Docker sítě `riot_default` |

Při lokálním vývoji nebo diagnostice jsou porty hlavního RIoT stacku popsány v README hlavního repozitáře.

## Reálné Zdroje Dat

Repozitář obsahuje preprocesory pro tři konkrétní dopravní zdroje:

- Waze Partner feed - dopravní zácpy a omezení nad silničními segmenty.
- NDIC DATEX II - dopravní stavy a FCD hodnoty s doplněním TMC lokalizačních metadat.
- IDS JMK/MHD - živé polohy vozidel z WebSocket streamu kombinované s GTFS jízdními řády.

Adresy zdrojů a přístupové údaje jsou nastavované přes `.env`. Pokud proměnné nejsou vyplněné, některé preprocesory používají výchozí hodnoty definované přímo ve svém `src/config.go`.


## Vývojové Závislosti

Pro běžné spuštění celé sestavy stačí Docker a Docker Compose. Při lokálním vývoji jednotlivých částí se používají také:

- Go pro preprocesory a modul `commons`.
- Python pro DATEX Downloader.
- Hlavní repozitář RIoT, protože preprocesory publikují data do jeho RabbitMQ infrastruktury.

Přesné příkazy a konfigurace jednotlivých modulů jsou uvedené v jejich README.

## Licence

Projekt je licencovaný pod [MIT licencí](LICENSE.txt).
