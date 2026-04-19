/**
 * @File: resolve_trip_hash_handler.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
    "bufio"
    "fmt"
    "net/http"
    "os"
    "regexp"
    "strings"
    "sync"
)

var (
    resolveTripHashApiMap  map[string]string
    resolveTripHashHashMap map[string]string
    resolveTripHashOnce    sync.Once
)

// Registrace handleru.
func init() {
    http.HandleFunc("/gtfs/resolve-trip-hash", resolveTripHashHandler)
    fmt.Println("/gtfs/resolve-trip-hash endpoint registered")
}

// Funkce loadResolveTripHashData načte mapování lineid/routeid na trip_id z api.txt a mapování trip_id na hash z trip_key_map.csv.
func loadResolveTripHashData() {
    resolveTripHashApiMap = make(map[string]string)
    resolveTripHashHashMap = make(map[string]string)

    // Načte api.txt (lineid/routeid => trip_id)
    apiFile, err := os.Open("static_data/api.txt")
    if err != nil {
        fmt.Println("Cannot open api.txt:", err)
        return
    }
    defer apiFile.Close()

    scanner := bufio.NewScanner(apiFile)
    re := regexp.MustCompile(`(\d+)/(\d+)\s*=\s*(\d+)$`)

    for scanner.Scan() {
        line := scanner.Text()
        match := re.FindStringSubmatch(line)
        if len(match) == 4 {
            key := fmt.Sprintf("%s/%s", match[1], match[2])
            resolveTripHashApiMap[key] = match[3]
        }
    }

    // Načte trip_key_map.csv (hash => trip_id).
    csvFile, err := os.Open("trip_key_map.csv")
    if err != nil {
        fmt.Println("Cannot open trip_key_map.csv:", err)
        return
    }
    defer csvFile.Close()

    csvScanner := bufio.NewScanner(csvFile)
    for csvScanner.Scan() {
        line := csvScanner.Text()
        parts := strings.Split(line, ",")
        if len(parts) != 2 {
            continue
        }
        resolveTripHashHashMap[parts[1]] = parts[0]
    }
}

// Funkce resolveTripHashHandler zpracovává HTTP požadavek a vrací hash pro trip podle kombinace lineid/routeid.
func resolveTripHashHandler(w http.ResponseWriter, r *http.Request) {
    resolveTripHashOnce.Do(loadResolveTripHashData)

    lineID := r.URL.Query().Get("lineid")
    routeID := r.URL.Query().Get("routeid")

    if lineID == "" || routeID == "" {
        http.Error(w, "Missing lineid or routeid", http.StatusBadRequest)
        return
    }

    key := fmt.Sprintf("%s/%s", lineID, routeID)
    tripID, ok := resolveTripHashApiMap[key]
    if !ok {
        http.Error(w, "Trip ID not found", http.StatusNotFound)
        return
    }

    hash, ok := resolveTripHashHashMap[tripID]
    if !ok {
        http.Error(w, "Hash not found for trip ID", http.StatusNotFound)
        return
    }

    fmt.Fprint(w, hash)
}