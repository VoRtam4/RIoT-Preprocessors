# DATEX Downloader

DATEX Downloader je malá FastAPI služba pro příjem push zpráv z NDIC ve formátu DATEX II. Přijaté XML ověří, volitelně zkontroluje HTTP Basic autentizaci, uloží poslední zprávu do paměti a zároveň ji persistuje jako XML soubor. NDIC preprocesor si z této služby následně může vyzvednout poslední dostupný snímek.

Modul neprovádí doménové zpracování dopravních dat. Slouží jako vstupní mezivrstva mezi NDIC push zdrojem a preprocesorem, který už řeší převod DATEX struktury do interních dat systému.

## Spuštění

Samostatně lze modul spustit z jeho adresáře:

```bash
docker compose up -d datex-server
```

Docker image spouští `uvicorn app.main:app` a publikuje aplikaci na portu `8000`. Služba nezávisí na RabbitMQ ani Backend Core; slouží jako HTTP mezivrstva mezi NDIC push zdrojem a NDIC preprocesorem. Při lokálním běhu se adresář `data/` mapuje do kontejneru, aby se přijaté XML zprávy zachovaly i po restartu.

## Konfigurace

Hlavní proměnné prostředí používané modulem:

- `PORT`: port uvnitř kontejneru, výchozí hodnota je `8000`
- `DATEX_USERNAME`: uživatelské jméno pro HTTP Basic autentizaci příchozího NDIC push požadavku
- `DATEX_PASSWORD`: heslo pro HTTP Basic autentizaci příchozího NDIC push požadavku

Pokud `DATEX_USERNAME` i `DATEX_PASSWORD` zůstanou prázdné, příjem dat nevyžaduje přihlášení.

## Rozhraní

Základní lokální adresa:

- HTTP API: `http://localhost:8000`

Vybrané endpointy:

- `POST /datex-in`: příjem NDIC XML zprávy, podporuje i `Content-Encoding: gzip`
- `GET /api/latest`: poslední přijatá zpráva zabalená v JSON odpovědi
- `GET /api/latest.xml`: poslední přijatá zpráva přímo jako XML
- `GET /download/latest.xml`: stažení posledního persistovaného XML souboru
- `GET /healthz`: jednoduchá kontrola dostupnosti služby

Příklad odeslání XML zprávy:

```bash
curl -X POST http://localhost:8000/datex-in \
  -u "$DATEX_USERNAME:$DATEX_PASSWORD" \
  -H "Content-Type: application/xml" \
  --data-binary @message.xml
```

Příklad načtení poslední zprávy:

```bash
curl http://localhost:8000/api/latest.xml
```

## Struktura modulu

- `Dockerfile`: build FastAPI služby a spuštění přes Uvicorn
- `docker-compose.yml`: samostatné lokální spuštění služby
- `render.yaml`: konfigurace pro nasazení na Render
- `requirements.txt`: Python závislosti
- `app/main.py`: FastAPI aplikace, autentizace, endpointy a persistování XML
- `app/storage.py`: jednoduché thread-safe uložení poslední zprávy v paměti
- `app/parser.py`: pomocný XML parser
- `data/`: lokální adresář pro persistované DATEX XML zprávy
