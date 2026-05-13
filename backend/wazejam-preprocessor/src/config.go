/**
 * @file config.go
 * @brief Konfigurace WazeJam preprocesoru a jeho napojení na zdroj dat a RIoT.
 *
 * @author Dominik Vondruška
 * @author Vojtěch Hubáček
 * @ingroup riot_wazejam_preprocessor
 *
 * @par Autorský podíl
 * - Dominik Vondruška: základní konfigurační vazba Waze preprocesoru na zdroj dat a RIoT.
 * - Vojtěch Hubáček: úprava konfigurace pro segmentový model instancí a rozšířené publikované hodnoty.
 */
package main

import "os"

const defaultWazeURL = "https://www.waze.com/row-partnerhub-api/partners/16198912488/waze-feeds/9c8b4163-e3c2-436f-86b7-3db2058ce7a1?format=1"

type appConfig struct {
	WazeURL string
}

func loadConfig() appConfig {
	return appConfig{
		WazeURL: getEnv("WAZE_URL", defaultWazeURL),
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
