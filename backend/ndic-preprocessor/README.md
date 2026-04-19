# NDIC Preprocessor

**NDIC Preprocessor** je modul systému, který zpracovává real‑time data o dopravních podmínkách z **Národního dopravního informačního centra (NDIC)**.  
Pravidelně stahuje data z API, parsuje XML/JSON odpovědi, převádí je na standardizované parametry a odesílá výsledky do systému prostřednictvím RabbitMQ.

---

## Účel modulu
- Načítá data o **dopravním stavu** (Traffic Status), **průměrné rychlosti** (Traffic Speed) a **době jízdy** (Travel Time) z NDIC API.
- Transformuje tyto informace do podoby vhodné pro systém RIoT a mapuje je na **SD instance** (např. NDIC_TRAFFIC_<ID>).
- Publikuje:
  - **KPI požadavky** (KPIFulfillmentCheckRequest)
  - **Žádosti o registraci SD instancí**, pokud jsou zjištěny nové úseky.

---

## Datové toky
- **Vstup:**  
  - HTTP endpoint NDIC `http://80.211.200.65:8000/api/latest` (data ve formátu JSON nebo XML).
- **Výstup:**  
  - RabbitMQ fronty:
    - `KPIFulfillmentCheckRequestsQueue` – KPI požadavky pro zpracování v systému RIoT.
    - `SDInstanceRegistrationRequestsQueue` – registrace nových SD instancí.

---

## Hlavní funkce
- **fetchAndProcessNDICData** – stahuje data z API, zpracovává je a připravuje KPI i registrace.
- **generateKPIRequest** – vytváří KPI požadavek a odesílá ho do RabbitMQ.
- **generateSDInstanceRegistrationRequest** – registruje nové SD instance do systému.
- **determineSDInstanceScenario** – určuje, zda je SD instance známá, nepotvrzená, nebo nová.
- **checkForSetOfSDTypesUpdates / checkForSetOfSDInstancesUpdates** – sledují RabbitMQ fronty a udržují seznam SD typů a instancí aktuální.

---

## Struktura projektu
- **main.go** – hlavní program, nastavuje spojení s RabbitMQ, cyklicky spouští stahování NDIC dat.
- **fetchAndProcessNDICData()** – logika pro parsování XML a přípravu výstupních dat.

---

## Kontext použití
NDIC Preprocessor funguje jako **automatický sběrač dopravních dat** z NDIC a jejich převodník do jednotného formátu pro backend systému.  
Zajišťuje, že systém má k dispozici nejaktuálnější údaje o stavu dopravy v České republice a může je využívat pro monitorování, reporting a KPI vyhodnocování.
