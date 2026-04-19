# MyRTAlerts

Modulární systém pro monitorování dopravních otevřených dat v reálném čase (Brno). Integruje data z Waze, NDIC a MHD (IDS JMK) a umožňuje nastavovat alerty při překročení definovaných kritérií. Postaveno jako bakalářská práce Dominika Vondrušky na FIT VUT (2025).

## Architektura a služby
MyRTAlerts využívá jádro systému **RIoT** (vyhodnocování KPI a GraphQL API) a doplňuje je o nové komponenty:

**Nové služby:**
- `rta-frontend` – webová aplikace (React/TypeScript)
- `rta-gtfs-backend`, `rta-tmc-backend` – služby pro GTFS a TMC (Go)
- `rta-ndic-preprocessor`, `rta-wazejam-preprocessor`, `rta-mhd-preprocessor` – preprocesory dat (Go)

**Převzaté z RIoT:**
- `riot-backend-core` – GraphQL API a logika KPI
- `riot-message-processing-unit` – stream processing
- `commons` – sdílené modely

## Požadavky
- **Docker** 20.10+
- **Docker Compose** 2.12+

## Porty
- Frontend: **3000**
- Backend API: **9090**
- GTFS: **9100**, TMC: **9200**
- DB (PostgreSQL): **5432**
- RabbitMQ: **5672**, web UI: **15672**
- (volitelně) Mosquitto: **1883**, **9001**

## Spuštění
```bash
docker-compose up --build -d
```
Frontend: [http://localhost:3000](http://localhost:3000)  

## Ukončení
```bash
docker-compose down
```

## Nasazené prostředí pro testovací účely

V rámci testování pro ověření funkcionality v rámci BP byl systém nasazen na VPS, kde je možné systém testovat na adrese:[http://http://194.182.85.5:3000/](http://http://194.182.85.5:3000/).  

## Původní systém RIoT
Projekt vychází z open-source platformy **RIoT** (Michal Bureš, FIT VUT), která poskytla základní jádro pro zpracování dat a vyhodnocování KPI. MyRTAlerts rozšiřuje RIoT o nové preprocesory a frontend pro dopravu v Brně.

## Licence
MIT License
