# NDIC Preprocessor

**NDIC Preprocessor** zpracovává DATEX II snapshoty uložené službou `datex-downloader`, převádí je do interního stavového modelu systému RIoT a publikuje výsledky přes RabbitMQ.  
Součástí zpracování je i enrichment z lokálně přibalených TMC tabulek, takže modul nevyžaduje samostatnou službu `tmc-core`.

---

## Účel modulu
- Periodicky načítá poslední uložený NDIC DATEX snapshot z `datex-downloader`.
- Parsuje veličiny **Traffic Status**, **Traffic Speed** a **Travel Time** pro `anyVehicle`.
- Vytváří a průběžně aktualizuje SD instance ve tvaru `NDIC_TRAFFIC_<sourceIdentification>`.
- Doplňuje stabilní lokalizační metadata z TMC, například lokalizační kód, název bodu, oblast a silnici.
- Publikuje stavy do front `KPIFulfillmentCheckRequestsQueue` a registrace do `SDInstanceRegistrationRequestsQueue`.

---

## Datové toky
- **Vstup:**  
  - `datex-downloader` endpoint `/api/latest` nebo `/api/latest.xml`
  - lokálně přibalená referenční TMC data v adresáři `static_data/tmc`
- **Výstup:**  
  - RabbitMQ fronty:
    - `KPIFulfillmentCheckRequestsQueue` – KPI požadavky pro zpracování v systému RIoT.
    - `SDInstanceRegistrationRequestsQueue` – registrace nových SD instancí.

---

## Hlavní funkce
- **fetchAndProcessNDICData** – načte poslední snapshot z downloaderu, spustí parsing a enrichment a publikuje výsledky.
- **parseNDICXML** – převádí DATEX II XML do interní struktury `parsedFetch`.
- **tmcEnricher** – načítá TMC body z lokálních referenčních souborů a doplňuje lokalizační metadata do snapshotů.
- **processFetchResult** – spravuje životní cyklus aktivních a neaktivních instancí.
- **registerSDType / registerInstanceIfNeeded** – registrace typu a instancí v novém systému.

---

## Struktura projektu
- **main.go** – start služby a plánování periodického zpracování.
- **fetch.go** – načítání posledního snapshotu z `datex-downloader`.
- **parser.go** – parsování DATEX II XML a extrakce stavových i lokalizačních dat.
- **tmc.go** – enrichment snapshotů o metadata z lokálně přibalených TMC dat.
- **runtime.go / publish.go / sdtype.go** – práce s instancemi a publikace do systému.

---

## Kontext použití
NDIC Preprocessor představuje stavovou integrační vrstvu mezi uloženými DATEX zprávami a zbytkem platformy RIoT.  
`datex-downloader` řeší pouze příjem a archivaci push zpráv, zatímco samotný preprocesor zajišťuje interpretaci, TMC enrichment, registraci entit a publikaci do interní komunikační vrstvy.
