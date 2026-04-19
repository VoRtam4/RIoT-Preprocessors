/**
 * @File: find_TS_locations.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: Modul pro procházení a vyhledávání v XML dokumentu předdefinovaných vozidel.
 */

package data

import (
    "encoding/xml"
    "io"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

// TSLocation představuje jednu TS lokaci ve výstupním JSONu.
type TSLocation struct {
    Ts    string `json:"ts"`
    Start Coord  `json:"start"`
    End   Coord  `json:"end"`
}

// Coord obsahuje zeměpisné souřadnice.
type Coord struct {
    Lat float64 `json:"lat"`
    Lon float64 `json:"lon"`
}

// lcdFilter páruje jeden LCD kód s požadovaným směrem.
type lcdFilter struct {
    code      int    // LCD kód
    direction string // "positive", "negative" nebo "" (oba)
}

// LCDFilter definuje filtr: konkrétní LCD kód + (volitelný) směr.
// Pokud je Direction == "", bere se oba směry.
type LCDFilter struct {
    Code      int
    Direction string
}

// FindTSLocations provádí streamové zpracování XML a vrací TS lokality, které odpovídají alespoň jednomu z filtrů.
// baseDir je cesta k adresáři obsahujícímu "pls_FCD_241201_LiteVersion.xml".
func FindTSLocations(filters []LCDFilter, baseDir string) ([]TSLocation, error) {
    fpath := filepath.Join(baseDir, "pls_FCD_241201_LiteVersion.xml")
    file, err := os.Open(fpath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    dec := xml.NewDecoder(file)
    var results []TSLocation
    seen := make(map[string]bool)

Outer:
    for {
        tok, err := dec.Token()
        if err != nil {
            // Konec souboru nebo neočekávaný EOF v XML → ukončíme parsování
            if err == io.EOF {
                break Outer
            }
            if serr, ok := err.(*xml.SyntaxError); ok && strings.Contains(serr.Error(), "unexpected EOF") {
                break Outer
            }
            return nil, err
        }
        se, ok := tok.(xml.StartElement)
        if !ok || se.Name.Local != "predefinedLocationContainer" {
            continue
        }

        var tsID string
        var primary, secondary int
        var dirCode string
        var startCoord, endCoord Coord
        var inName, inPrimary, inSecondary, inLRP, inLastLRP bool

    Inner:
        for {
            innerTok, err := dec.Token()
            if err != nil {
                // Konec elementu nebo neočekávaný EOF v XML → ukončíme vnitřní smyčku
                if err == io.EOF {
                    break Inner
                }
                if serr, ok := err.(*xml.SyntaxError); ok && strings.Contains(serr.Error(), "unexpected EOF") {
                    break Inner
                }
                return nil, err
            }

            switch t := innerTok.(type) {
            case xml.StartElement:
                switch t.Name.Local {
                case "predefinedLocationName":
                    inName = true
                case "value":
                    if inName {
                        var v string
                        dec.DecodeElement(&v, &t)
                        tsID = strings.TrimSpace(v)
                    }
                case "alertCMethod2PrimaryPointLocation":
                    inPrimary = true
                case "alertCMethod2SecondaryPointLocation":
                    inSecondary = true
                case "specificLocation":
                    var locStr string
                    dec.DecodeElement(&locStr, &t)
                    code, _ := strconv.Atoi(strings.TrimSpace(locStr))
                    if inPrimary {
                        primary = code
                    }
                    if inSecondary {
                        secondary = code
                    }
                case "alertCDirectionCoded":
                    var d string
                    dec.DecodeElement(&d, &t)
                    dirCode = strings.TrimSpace(d)
                case "openlrLocationReferencePoint":
                    inLRP = true
                case "openlrLastLocationReferencePoint":
                    inLastLRP = true
                case "latitude":
                    var latStr string
                    dec.DecodeElement(&latStr, &t)
                    lat, _ := strconv.ParseFloat(strings.TrimSpace(latStr), 64)
                    if inLRP {
                        startCoord.Lat = lat
                    }
                    if inLastLRP {
                        endCoord.Lat = lat
                    }
                case "longitude":
                    var lonStr string
                    dec.DecodeElement(&lonStr, &t)
                    lon, _ := strconv.ParseFloat(strings.TrimSpace(lonStr), 64)
                    if inLRP {
                        startCoord.Lon = lon
                    }
                    if inLastLRP {
                        endCoord.Lon = lon
                    }
                }
            case xml.EndElement:
                switch t.Name.Local {
                case "predefinedLocationName":
                    inName = false
                case "alertCMethod2PrimaryPointLocation":
                    inPrimary = false
                case "alertCMethod2SecondaryPointLocation":
                    inSecondary = false
                case "openlrLocationReferencePoint":
                    inLRP = false
                case "openlrLastLocationReferencePoint":
                    inLastLRP = false
                case "predefinedLocationContainer":
                    // Konec kontejneru → zpracujeme filtr
                    for _, f := range filters {
                        if (f.Code == primary || f.Code == secondary) &&
                            (f.Direction == "" || strings.EqualFold(dirCode, f.Direction)) {
                            if tsID != "" && !seen[tsID] {
                                results = append(results, TSLocation{Ts: tsID, Start: startCoord, End: endCoord})
                                seen[tsID] = true
                            }
                            break
                        }
                    }
                    break Inner
                }
            }
        }
    }

    return results, nil
}
