# TMC-Core

**TMC-Core** je samostatný modul napsaný v jazyce Go, který zajišťuje načítání a poskytování dat z lokalizační tabulky TMC (Traffic Message Channel).  
Umožňuje vyhledávání lokalit, překlad názvů míst na TMC kódy a poskytuje hierarchii silnic a administrativních oblastí pro využití v systému monitorování dopravy.

---

## Účel modulu
- Načítání TMC lokalizačních tabulek (silnice, segmenty, body, oblasti).  
- Vyhledávání a překlad lokalit na **TS kódy**.  
- Poskytování dat ostatním částem systému přes HTTP API.

---

## Funkce
- **Fulltextové vyhledávání** lokalit (např. názvů měst, úseků, křižovatek).  
- **Zobrazení stromu silnic a segmentů.**  
- **Překlad předdefinovaných poloh na kódy**.  
- **Vyhledávání administrativních oblastí a přiřazených dat.**

---

## HTTP endpointy

| Endpoint | Popis | Příklad volání |
|---------|------|----------------|
| **`/lcd`** | Vrátí kompletní seznam všech LCD záznamů. | `/lcd` |
| **`/lcd?q=<dotaz>`** | Fulltext vyhledávání LCD podle názvu (např. ulice, místo). | `/lcd?q=Olomoucká` |
| **`/lcd/admins?lcd=<id>`** | Vrátí administrativní oblasti (obce, kraje) příslušné k LCD. | `/lcd/admins?lcd=33294` |
| **`/lcd/roads?lcd=<id>`** | Vrátí silnice spojené s danými LCD. | `/lcd/roads?lcd=43369` |
| **`/lcd/admins-roads?admins=<ids>&roads=<ids>`** | Kombinace – vrací body dle LCD získaných z admins i roads. | `/lcd/admins-roads?admins=260,261&roads=574` |
| **`/roads?area_ref=<id>`** | Seznam silnic (a jejich segmentů) pro danou lokalitu. | `/roads?area_ref=10` |
| **`/admins?roads=<ids>`** | Vrátí administrativní oblasti, kterými vedou uvedené silnice. | `/admins?roads=574,25036` |
| **`/points?lcd=<id>`** | Vrací body podle LCD (kombinovatelné s roads/admins). | `/points?lcd=37185` |
| **`/predefined_positions?lcd=<ids>&direction=<dir>`** | Vrací předdefinované polohy (TS body) pro vybrané LCD a směr. | `/predefined_positions?lcd=4869,4870,17621&direction=negative` |

---

## Struktura modulu
- **main.go** – spouštěcí bod aplikace, registrace endpointů.  
- **handlers/** – definice HTTP handlerů.  
- **data/** – načítání a správa TMC tabulek.  
- **static_data/** – uložené textové soubory TMC (roads.txt, admins.txt atd.).

---

## Technická specifikace
- **Jazyk:** Go (Golang)  
- **Formát komunikace:** REST API (HTTP/JSON)  
- **Datové zdroje:** TMC lokalizační tabulka (body, silnice, segmenty, admins)

---

## Kontext
Modul funguje jako samostatná služba a poskytuje vyhledávací a překladovou funkcionalitu pro dopravní systém založený na TMC datech. Je využíván dalšími částmi systému, zejména pro mapování dopravních událostí na konkrétní úseky a oblasti.
