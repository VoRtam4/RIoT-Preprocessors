# Datex Downloader – Push přijímač NDIC zpráv

Tento modul slouží jako jednoduchá vstupní brána pro push zprávy z NDIC ve formátu DATEX II.  
Jeho úkolem je zprávu přijmout, ověřit, uložit a zpřístupnit poslední snapshot modulu `ndic-preprocessor`.

---

## Účel modulu
Datex Downloader poskytuje rozhraní pro příjem zpráv od NDIC, ověřuje jejich správnost,
ukládá je do souborového úložiště a vystavuje poslední přijatý snapshot přes jednoduché HTTP API.
Neprovádí vlastní parsování dopravních dat ani pull akvizici z NDIC.

---

## Použité technologie
- **Python 3.11+** – implementace serveru.
- **FastAPI** – webový framework pro HTTP endpointy.
- **uvicorn** – ASGI server pro běh aplikace.
- **xml.etree.ElementTree** – zpracování a validace XML dat.
- **gzip** – dekomprese příchozích zpráv.

---

## Funkcionalita
- Příjem zpráv NDIC přes endpoint `/datex-in` s podporou **HTTP Basic autentizace**.
- Ověření a uložení zpráv ve formátu XML do adresáře `ndic_messages`.
- Poskytování poslední zprávy jako JSON wrapper nebo přímo jako XML.
- Udržování omezeného počtu archivovaných snapshotů na disku.

---

## Struktura projektu
- **main.py** – hlavní aplikace FastAPI se všemi endpointy.
- **storage.py** – jednoduché in-memory uložení poslední zprávy.
- **requirements.txt** – přehled potřebných Python knihoven.

---

## Hlavní endpointy

| Endpoint                 | Metoda  | Popis |
|--------------------------|--------|------|
| `/datex-in`              | POST   | Přijímá NDIC zprávy, ověřuje autentizaci, zpracovává a ukládá zprávy. |
| `/api/latest`            | GET    | Vrací poslední zprávu ve formátu JSON. |
| `/api/latest.xml`        | GET    | Vrací poslední zprávu ve formátu XML. |
| `/download/latest.xml`   | GET    | Vrací poslední uloženou zprávu jako XML soubor. |
| `/healthz`               | GET    | Jednoduchý health endpoint. |

---

# Nasazení a spuštění

### Příprava prostředí
Projekt je připraven pro nasazení pomocí **Docker Compose**.  
Stačí mít nainstalovaný **Docker** a **Docker Compose**.

### Spuštění aplikace
```bash
docker-compose up --build -d
```

- `--build` zajistí, že se při spuštění přegeneruje image (vhodné při změnách v kódu).  
- `-d` spustí aplikaci v „detached“ režimu (na pozadí).

Po spuštění bude aplikace dostupná na adrese:
```
http://localhost:8000
```
*(Port lze upravit v souboru `docker-compose.yml`, pokud je potřeba jiný).*

### Zastavení aplikace
```bash
docker-compose down
```

---

## Kontext použití
Tento modul funguje jako záměrně jednoduchá mezivrstva mezi NDIC a `ndic-preprocessor`.  
Přijímá push zprávy, archivuje je a vystavuje poslední snapshot tak, aby samotný preprocesor mohl zůstat stavový a soustředit se na parsing, enrichment a integraci do systému.
