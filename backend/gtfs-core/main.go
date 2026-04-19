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
	"github.com/rs/cors"
)


func main() {
	// Update static_data.
	go scheduleWeeklyUpdate()
	http.HandleFunc("/force-update", handleForceUpdate)
	// Všechno registrují jednotlivé init() funkce v souborech
	load()
	// Nastavení CORS policy.
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(http.DefaultServeMux)

	log.Println("Server listening on :9100")
	if err := http.ListenAndServe(":9100", handler); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
