# GTFS-Core

Modul **GTFS-Core** představuje specializovanou komponentu systému pro zpracování a poskytování dat ve formátu **GTFS (General Transit Feed Specification)**.  
Je implementován v jazyce **Go** a zajišťuje pravidelný sběr, aktualizaci, zpracování a zpřístupnění statických dat veřejné dopravy prostřednictvím rozhraní HTTP API.  

---

## Účel modulu

GTFS-Core slouží jako centrální uzel pro práci se statickými daty hromadné dopravy. Jeho hlavními cíli jsou:

- Automatizace aktualizace GTFS dat – modul stahuje aktuální datový balíček ve formátu `gtfs.zip` z oficiálního zdroje KORDIS JMK, rozbaluje jej a připravuje data pro další zpracování.  
- Generování pomocných map a klíčů – zejména souboru `trip_key_map.csv`, který propojuje konkrétní spoje (`trip_id`) se stabilními identifikátory.  
- Zpřístupnění funkčnosti přes HTTP endpointy – pro ostatní části systému i externí nástroje.  
- Integrace s RabbitMQ – registrace chybějících instancí sledovaných spojů (SD instancí) do distribuční platformy RIoT.

---

## Funkční rozsah

### 1. Aktualizace dat
- Automatická – naplánovaná jednou týdně (neděle).
- Manuální – spuštění přes speciální endpoint `/force-update`.

### 2. Zpracování GTFS dat
- Rozbalení dat do adresáře `static_data`.
- Vygenerování souboru `trip_key_map.csv` (mapa spojů a stabilních klíčů).
- Uložení časové známky poslední aktualizace (`last_update.txt`).

### 3. Poskytované služby (HTTP API)
- `/gtfs/departure-times` – vrací časy odjezdů spojů pro zadanou linku, zastávky a den.
- `/gtfs/resolve-stop-id` – vyhledá všechny varianty `stop_id` pro konkrétní zastávku.
- `/gtfs/resolve-trip-hash` – převede kombinaci `lineid`/`routeid` na stabilní hash spoje.
- `/gtfs/resolve-trip-id` – získá `trip_id` podle kombinace linky, zastávek a směru.
- `/gtfs/resolve-trip-details` – vrátí detailní informace o konkrétním tripu (číslo linky, název zastávek, dny provozu).
- `/gtfs/stable-trip-key` – vypočítá stabilní klíč (SHA-256) na základě parametrů spoje.
- `/gtfs/stops` – vrací seznam unikátních zastávek linky v pořadí jejich průjezdu.
- `/gtfs/routes` – vrací seznam všech linek (route_id, číslo linky, typ dopravy).
- `/force-update` – spustí manuální aktualizaci GTFS dat (stažení, rozbalení, registrace).



### 4. Registrace SD instancí
Každý nový spoj je při aktualizaci zaregistrován do RabbitMQ fronty `SDInstanceRegistrationRequestsQueueName` pod identifikátorem ve formátu `MHD_TRIP_<hash>`.

---

## Technická specifikace

- **Programovací jazyk:** Go (Golang)  
- **Komunikační rozhraní:** REST API (HTTP/JSON)  
- **Datový zdroj:** GTFS feed KORDIS JMK (`https://kordis-jmk.cz/gtfs/gtfs.zip`)  
- **Integrace:** RabbitMQ – výměna zpráv o nových instancích dopravních spojů  

---

## Struktura modulu

- `main.go` – vstupní bod aplikace, inicializace endpointů a plánovače aktualizací.
- `static_data/` – adresář s rozbalenými GTFS soubory (`trips.txt`, `stops.txt`, `routes.txt` aj.).
- `trip_key_map.csv` – generovaná mapa stabilních klíčů a tripů.
- `last_update.txt` – časová známka poslední aktualizace dat.

---

## Proces aktualizace

1. Stažení GTFS datového balíčku (`gtfs.zip`).  
2. Rozbalení obsahu do adresáře `static_data`.  
3. Generování mapy spojů (`trip_key_map.csv`).  
4. Registrace chybějících SD instancí do RabbitMQ.  
5. Zápis časové známky poslední aktualizace.

---

## HTTP endpointy

| Endpoint                     | Metoda | Parametry                                             | Popis                                                                 |
|------------------------------|--------|-------------------------------------------------------|----------------------------------------------------------------------|
| `/gtfs/departure-times`      | GET    | `route_id`, `from_stop`, `to_stop`, `day`, `direction` | Vrátí seznam časů odjezdů spojů mezi zadanými zastávkami v daný den. |
| `/gtfs/resolve-stop-id`      | GET    | `base_stop_id`                                        | Vrátí všechny stop_id varianty pro základní zastávku.                |
| `/gtfs/resolve-trip-hash`    | GET    | `lineid`, `routeid`                                   | Vrátí hash tripu odpovídající kombinaci linky a trasy.               |
| `/gtfs/resolve-trip-id`      | GET    | `route_id`, `from_stop`, `to_stop`, `direction`       | Najde `trip_id` podle zadaných parametrů.                            |
| `/gtfs/resolve-trip-details` | GET    | `trip_id`                                             | Vrátí detailní informace o konkrétním tripu (linka, zastávky, dny).  |
| `/gtfs/stable-trip-key`      | GET    | `route_id`, `from_stop`, `to_stop`, `day`, `departure_time` | Vypočítá stabilní klíč pro konkrétní trip.                          |
| `/gtfs/stops`                | GET    | `route_id`                                            | Vrátí seznam unikátních zastávek linky seřazený podle pořadí.        |
| `/gtfs/routes`               | GET    | —                                                     | Vrátí seznam všech linek (route_id, číslo linky, typ dopravy).       |
| `/force-update`              | GET    | —                                                     | Spustí manuální aktualizaci GTFS dat (stažení, rozbalení, registrace).|
---

## Licence a kontext použití

Modul je součástí bakalářské práce „Systém pro monitorování otevřených dat v reálném čase“.  
Je určen pro akademické a výzkumné účely, zejména pro integraci otevřených dat veřejné dopravy do systému RIoT.  
