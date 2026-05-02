package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
)

func fetchAndProcessNDICData(client rabbitmq.Client, config appConfig, enricher *tmcEnricher) time.Time {
	fetch, err := fetchNDIC(config)
	if err != nil {
		log.Printf("[NDIC] Fetch failed: %v", err)
		return time.Time{}
	}

	if enricher != nil {
		enricher.enrichFetch(fetch)
	}

	log.Printf("[NDIC] Processing publication %s with %d snapshots", fetch.PublicationTime.Format(time.RFC3339), len(fetch.Snapshots))
	processFetchResult(client, config, fetch)
	return fetch.PublicationTime
}

func fetchNDIC(config appConfig) (*parsedFetch, error) {
	resp, err := http.Get(config.NDICURL)
	if err != nil {
		return nil, fmt.Errorf("download NDIC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download NDIC: status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read NDIC body: %w", err)
	}

	xmlBytes, err := unwrapXML(body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	parsed, err := parseNDICXML(xmlBytes)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func unwrapXML(body []byte, contentType string) ([]byte, error) {
	if strings.Contains(contentType, "application/json") {
		var wrapper fetchEnvelope
		if err := json.Unmarshal(body, &wrapper); err != nil {
			return nil, fmt.Errorf("parse NDIC JSON wrapper: %w", err)
		}
		return []byte(wrapper.LatestRaw), nil
	}
	return body, nil
}
