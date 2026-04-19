/**
 * @File: main.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"tmc-core/src/data"
	"tmc-core/src/handlers"
)

func main() {
	// Cesty ke statickým datům.
	tmcDir := os.Getenv("TMC_TMC_DIR")
	if tmcDir == "" {
		tmcDir = "/app/static_data/tmc"
	}

	// Načtení TMC dat (např. points.txt, roads.txt, segments.txt).
	loader := data.NewLoader(tmcDir)
	if err := loader.Load(); err != nil {
		log.Fatalf("Nepodařilo se načíst data: %v", err)
	}

	// Načtení admin lokalit pro endpoint /lcd/admins.
	if err := data.LoadAdminLocations(tmcDir); err != nil {
		log.Fatalf("Nepodařilo se načíst admin lokality: %v", err)
	}

	// Načtení road a segments cest pro endpoint /roads.
	if _, err := data.BuildRoadList(tmcDir, "1"); err != nil {
		log.Fatalf("Chyba při načítání silnic: %v", err)
	}

	// Inicializace HTTP serveru bez default middleware.
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Registrace endpointů.
	handlers.RegisterLCD(router, loader)

	// Nastavení CORS policy.
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Obalíme router CORS handlerem.
	handler := c.Handler(router)

	// Spuštění serveru přes http.ListenAndServe s handlerem CORS.
	addr := ":9200"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	log.Printf("Spouštím tmc-core na %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Chyba při spuštění serveru: %v", err)
	}
}
