/**
 * @File: routes_handler.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package main

import (
	"net/http"
	"fmt"
	"encoding/json"
)

func routesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

func init() {
	http.HandleFunc("/gtfs/routes", routesHandler)
	fmt.Println("/gtfs/routes endpoint registered")
}