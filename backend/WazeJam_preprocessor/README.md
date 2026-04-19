# WAZE Jam Preprocessor

**WAZE Jam Preprocessor** je modul systému, který zpracovává real‑time dopravní data z platformy **Waze**.  
Pravidelně stahuje JSON feed Waze, analyzuje jednotlivé dopravní události a převádí je na SD instance systému.  
Díky tomu systém získává přehled o dopravních zácpách, nepravidelnostech a upozorněních hlášených komunitou Waze.

---

## Účel modulu
- **Stahování dat z Waze API** – každé 2 minuty se provede dotaz na Waze Partner Feed.  
- **Převod dat na SD instance** – pro každou novou událost se vytvoří SD instance (UID) ve formátu: WAZE_JAM_<město>_<ulice>
- **Registrace SD instancí** – pokud událost dosud nebyla v systému známa, modul ji zaregistruje (SDInstanceRegistrationRequest).  
- **Publikování KPI požadavků** – pro všechny aktuální i ukončené události se odesílají KPI zprávy do systému.


## Hlavní funkce
- **fetchAndProcessWazeData** – stahuje data z Waze API, parsuje je a zpracovává jednotlivé události.
- **generateKPIRequest** – vytváří KPI požadavek a odesílá ho do RabbitMQ.
- **generateSDInstanceRegistrationRequest** – registruje nové SD instance do systému.
- **determineSDInstanceScenario** – určuje, zda je SD instance známá, nepotvrzená, nebo nová.
- **sanitizeString** – čistí názvy měst a ulic, aby byly použitelné pro generování UID.

---

## Datové toky
- **Vstup:**  
- Waze Partner Feed (JSON API):  
  `https://www.waze.com/row-partnerhub-api/partners/16198912488/waze-feeds/9c8b4163-e3c2-436f-86b7-3db2058ce7a1?format=1`
- **Výstup:**  
- RabbitMQ fronty:
  - `KPIFulfillmentCheckRequestsQueue` – KPI požadavky.
  - `SDInstanceRegistrationRequestsQueue` – registrace nových SD instancí.

---

## Logika zpracování
1. Modul stáhne data z Waze API (sekce `jams`, `alerts`, `irregularities`).  
2. Každá událost obsahuje UUID, město a ulici:
 - vytvoří se UID ve formátu `WAZE_JAM_<město>_<ulice>`.
3. Pokud událost ještě není v systému známá:
 - odešle se **registrace SD instance**.
4. Pro všechny události (nové i stávající):
 - odešle se **KPI požadavek** s detaily (uuid, city, street a další parametry).
5. Pokud událost zmizí z feedu:
 - odešle se KPI s nulovými hodnotami (delay=0, length=0, level=0),
 - událost se odstraní z lokálního seznamu.

---

## Struktura projektu
- **main.go** – hlavní program, nastavuje RabbitMQ, plánuje stahování a zpracování dat.
- **fetchAndProcessWazeData()** – logika zpracování JSON feedu.

---

## Kontext použití
WAZE Jam Preprocessor slouží jako **real‑time most** mezi daty z aplikace Waze a backendem systému.  
Systém díky němu získává okamžitý přehled o dopravních událostech hlášených uživateli Waze a dokáže je využít pro KPI vyhodnocování, notifikace a predikce dopravy.