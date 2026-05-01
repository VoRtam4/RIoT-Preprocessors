# MHD Preprocessor

**MHD Preprocessor** je modul systému, který zpracovává real‑time data o poloze vozidel městské hromadné dopravy v Brně.  
Preferuje živý WebSocket stream, při jeho výpadku nebo delší neaktivitě automaticky přepne na REST polling, data dál páruje s GTFS a odesílá do systému zprávy o KPI a registraci SD instancí.

---

## Účel modulu
- Připojuje se na **WebSocket města Brna** a odebírá data o aktuálních polohách vozidel MHD (tramvaje, autobusy, trolejbusy).  
- **Mapuje data na SD instance** systému (např. MHD_TRIP_<hash>) a rozhoduje, zda je instance již potvrzená či nová.  
- Odesílá **KPI požadavky** a v případě nových spojů také **žádosti o registraci SD instancí** do RabbitMQ.  

---

## Hlavní funkce
- **WebSocket listener** – kontinuálně přijímá polohová data z endpointu GIS Brno.  
- **REST polling fallback** – pokud WebSocket nepřináší data nebo je nedostupný, modul přejde na polling ArcGIS FeatureServeru a stejná data žene stejnou zpracovatelskou pipeline.  
- **Vestavěný GTFS enricher** – periodicky stahuje GTFS archiv, sestavuje interní definice spojů a zajišťuje stabilní identifikaci jízd bez potřeby samostatné služby `gtfs-core`.  
- **RabbitMQ komunikace** – odesílá KPI a registrační zprávy, přijímá aktualizace seznamů SD typů a instancí.  

---

## Datové toky
- **Vstup:** preferovaný WebSocket stream `wss://gis.brno.cz/geoevent/...` a fallback REST polling ArcGIS `FeatureServer/query`.  
- **Výstup:** RabbitMQ fronty:
  - `KPIFulfillmentCheckRequestsQueue` – KPI požadavky.  
  - `SDInstanceRegistrationRequestsQueue` – registrace nových SD instancí.

---

## Struktura projektu
- **main.go** – hlavní soubor, nastavuje spojení (WebSocket, polling fallback, RabbitMQ) a spouští zpracování.  
- **processWebSocketMessage / poll_source.go** – logika pro přijetí zpráv z WebSocketu i pollingu a jejich převedení na společný interní záznam.  
- **gtfs_store.go** – lokální načítání a parsování GTFS dat a sestavení stabilních definic jízd.  

---

## Konfigurace fallbacku
- `MHD_WS_URL` – volitelný override preferovaného WebSocket endpointu.
- `MHD_POLLING_URL` – volitelný override REST endpointu pro polling fallback.
- `MHD_GTFS_URL` – volitelný override GTFS archivu.

Časování fallbacku, polling interval, reconnect delay a ostatní režijní hodnoty jsou záměrně definované jen v interním configu preprocessoru jako konstanty v `config.go`, ne přes ENV.

---

## Kontext použití
MHD Preprocessor funguje jako **real‑time most** mezi živým datovým tokem MHD a backendem.  
Díky vlastnímu zpracování GTFS i živých WebSocket zpráv může modul fungovat samostatně bez odděleného GTFS modulu a systém tak získává v reálném čase informace o provozu MHD pro vyhodnocování KPI a správu SD instancí.
