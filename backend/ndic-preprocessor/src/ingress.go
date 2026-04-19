package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
)

var (
	latestRawXML   string
	latestRawMutex sync.RWMutex
)

func startHTTPServer(client rabbitmq.Client, config appConfig) {
	if config.RawStorageDir != "" {
		if err := os.MkdirAll(config.RawStorageDir, 0o755); err != nil {
			log.Printf("[NDIC] Failed to create raw storage dir %s: %v", config.RawStorageDir, err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/datex-in", func(w http.ResponseWriter, r *http.Request) {
		handleDatexIn(w, r, client, config)
	})
	mux.HandleFunc("/api/latest", handleGetLatest)
	mux.HandleFunc("/api/latest.xml", handleGetLatestXML)
	mux.HandleFunc("/download/latest.xml", func(w http.ResponseWriter, r *http.Request) {
		handleDownloadLatestXML(w, r)
	})

	log.Printf("[NDIC] HTTP ingress listening on %s", config.HTTPListenAddr)
	if err := http.ListenAndServe(config.HTTPListenAddr, mux); err != nil {
		log.Fatalf("[NDIC] HTTP server failed: %v", err)
	}
}

func handleDatexIn(w http.ResponseWriter, r *http.Request, client rabbitmq.Client, config appConfig) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !authorized(r, config) {
		w.Header().Set("WWW-Authenticate", `Basic realm="ndic"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rawData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gzipReader, err := gzip.NewReader(bytes.NewReader(rawData))
		if err != nil {
			http.Error(w, "failed to decompress gzip data", http.StatusBadRequest)
			return
		}
		decompressed, err := io.ReadAll(gzipReader)
		_ = gzipReader.Close()
		if err != nil {
			http.Error(w, "failed to decompress gzip data", http.StatusBadRequest)
			return
		}
		rawData = decompressed
	}

	fetch, err := parseNDICXML(rawData)
	if err != nil {
		http.Error(w, "invalid NDIC XML", http.StatusBadRequest)
		return
	}

	saveLatestRaw(string(rawData))
	persistRawXML(config.RawStorageDir, rawData, fetch.PublicationTime)
	processFetchResult(client, config, fetch)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGetLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	latest, ok := getLatestRaw()
	if !ok {
		http.Error(w, "no data available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"latest_raw": latest})
}

func handleGetLatestXML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	latest, ok := getLatestRaw()
	if !ok {
		http.Error(w, "no data available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(latest))
}

func handleDownloadLatestXML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	latest, ok := getLatestRaw()
	if !ok {
		http.Error(w, "no data available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", `attachment; filename="latest.xml"`)
	_, _ = w.Write([]byte(latest))
}

func authorized(r *http.Request, config appConfig) bool {
	if config.BasicAuthUsername == "" && config.BasicAuthPassword == "" {
		return true
	}

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Basic ") {
		return false
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return false
	}

	username, password, found := strings.Cut(string(raw), ":")
	if !found {
		return false
	}
	return username == config.BasicAuthUsername && password == config.BasicAuthPassword
}

func saveLatestRaw(raw string) {
	latestRawMutex.Lock()
	latestRawXML = raw
	latestRawMutex.Unlock()
}

func getLatestRaw() (string, bool) {
	latestRawMutex.RLock()
	defer latestRawMutex.RUnlock()
	if latestRawXML == "" {
		return "", false
	}
	return latestRawXML, true
}

func persistRawXML(dir string, raw []byte, publicationTime time.Time) {
	if dir == "" {
		return
	}

	fileTime := publicationTime
	if fileTime.IsZero() {
		fileTime = time.Now().UTC()
	}

	filename := fmt.Sprintf("%s.xml", fileTime.Format("2006-01-02T15-04-05Z07-00"))
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		log.Printf("[NDIC] Failed to store raw XML %s: %v", path, err)
	}
}
