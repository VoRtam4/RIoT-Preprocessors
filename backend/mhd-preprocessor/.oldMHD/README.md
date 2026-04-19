# MHD Preprocessor

**MHD Preprocessor** je modul systému, který zpracovává real‑time data o poloze vozidel městské hromadné dopravy v Brně.  
Přijímá stream dat z městského WebSocketu, vyhodnocuje je, páruje s GTFS daty a odesílá do systému zprávy o KPI a registraci SD instancí.

---

## Účel modulu
- Připojuje se na **WebSocket města Brna** a odebírá data o aktuálních polohách vozidel MHD (tramvaje, autobusy, trolejbusy).  
- **Mapuje data na SD instance** systému (např. MHD_TRIP_<hash>) a rozhoduje, zda je instance již potvrzená či nová.  
- Odesílá **KPI požadavky** a v případě nových spojů také **žádosti o registraci SD instancí** do RabbitMQ.  

---

## Hlavní funkce
- **WebSocket listener** – kontinuálně přijímá polohová data z endpointu GIS Brno.  
- **Integrace s GTFS Core** – pro získání stabilního hashe (`resolve-trip-hash`) k jednoznačné identifikaci spoje.  
- **RabbitMQ komunikace** – odesílá KPI a registrační zprávy, přijímá aktualizace seznamů SD typů a instancí.  

---

## Datové toky
- **Vstup:** WebSocket stream `wss://gis.brno.cz/geoevent/...` s daty o poloze MHD.  
- **Výstup:** RabbitMQ fronty:
  - `KPIFulfillmentCheckRequestsQueue` – KPI požadavky.  
  - `SDInstanceRegistrationRequestsQueue` – registrace nových SD instancí.

---

## Struktura projektu
- **main.go** – hlavní soubor, nastavuje spojení (WebSocket, RabbitMQ) a spouští zpracování.  
- **processWebSocketMessage** – logika pro zpracování zpráv (parsování atributů, příprava KPI, registrace SD instance).  
- **resolveStableTripHash** – dotaz na GTFS Core pro získání stabilního identifikátoru spoje.  

---

## Kontext použití
MHD Preprocessor funguje jako **real‑time most** mezi živým datovým tokem MHD a backendem.  
Díky tomu systém získává v reálném čase informace o provozu MHD a dokáže je využít k vyhodnocování KPI a správě SD instancí.
