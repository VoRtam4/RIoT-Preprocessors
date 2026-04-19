/**
 * @File: update_gtfs_data.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: Služba pro stažení a pravidelnou aktualizaci GTFS dat. Umožňuje také manuální update dat pomocí endpointu
 */

package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedConstants"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const gtfsURL = "https://kordis-jmk.cz/gtfs/gtfs.zip" //Záložní druhý zdroj z data.Brno: "https://www.arcgis.com/sharing/rest/content/items/379d2e9a7907460c8ca7fda1f3e84328/data"
const staticDataDir = "./static_data"
const localZipPath = "./gtfs_static.zip"
const maxAge = 7 * 24 * time.Hour // 7 days
const rabbitMQURL = "amqp://guest:guest@172.25.0.6:5672/"
const lastUpdateFile = "last_update.txt"

var updateRunning = false

var (
	sdTypes          = sharedUtils.NewSet[string]()
	sdTypesMutex     sync.Mutex
	sdInstances      = sharedUtils.NewSet[sharedModel.SDInstanceInfo]()
	sdInstancesMutex sync.Mutex
)

// Funkce registerMissingSDInstances projde trip_key_map.csv a zaregistruje chybějící SD instance do RabbitMQ.
func registerMissingSDInstances(client rabbitmq.Client) {
	file, err := os.Open("trip_key_map.csv")
	if err != nil {
		log.Printf("[ERROR] Cannot open trip_key_map.csv: %v", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[ERROR] Failed to read trip_key_map.csv: %v", err)
		return
	}

	count := 0

	for _, row := range records {
		if len(row) != 2 {
			continue
		}
		hash := row[0]
		uid := "MHD_TRIP_" + hash

		sdInstancesMutex.Lock()
		alreadyRegistered := sdInstances.Contains(sharedModel.SDInstanceInfo{
			SDInstanceUID: uid,
		})
		sdInstancesMutex.Unlock()

		if alreadyRegistered {
			continue
		}

		// Registrace SD instance.
		message := sharedModel.SDInstanceRegistrationRequestISCMessage{
			EventTime:     time.Now().UTC(),
			Label:         uid,
			SDInstanceUID: uid,
			SDTypeUID:     "MHD",
		}
		jsonMessage := sharedUtils.SerializeToJSON(message)
		if jsonMessage.IsFailure() {
			log.Printf("[ERROR] Failed to serialize SDInstance: %v", jsonMessage.GetError())
			continue
		}

		err := client.PublishJSONMessage(
			sharedUtils.NewEmptyOptional[string](),
			sharedUtils.NewOptionalOf(sharedConstants.SDInstanceRegistrationRequestsQueueName),
			jsonMessage.GetPayload(),
		)
		if err != nil {
			log.Printf("[ERROR] Failed to publish SDInstanceRegistrationRequest for %s: %v", uid, err)
			continue
		}
		log.Printf("[INFO] Registered missing SDInstance: %s", uid)
		count++
	}

	log.Printf("[INFO] Finished SDInstance sync. Newly registered: %d", count)
}

// Funkce handleForceUpdate spouští manuální aktualizaci GTFS dat přes HTTP endpoint.
func handleForceUpdate(w http.ResponseWriter, r *http.Request) {
	if updateRunning {
		fmt.Fprintln(w, "Update already in progress.")
		return
	}
	go performUpdate()
	fmt.Fprintln(w, "Manual update triggered.")
}

// Funkce scheduleWeeklyUpdate plánuje automatickou týdenní aktualizaci GTFS dat (každou neděli).
func scheduleWeeklyUpdate() {
	for {
		now := time.Now()
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, daysUntilSunday)
		duration := time.Until(next)
		log.Printf("Next update scheduled for: %s\n", next.Format(time.RFC1123))
		time.Sleep(duration)
		performUpdate()
		time.Sleep(7 * 24 * time.Hour) // wait 1 week
	}
}

// Funkce readLastUpdateTime načte datum poslední aktualizace z last_update.txt.
func readLastUpdateTime() (time.Time, error) {
	data, err := os.ReadFile(lastUpdateFile)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
}

func saveLastUpdateTime(t time.Time) error {
	return os.WriteFile(lastUpdateFile, []byte(t.Format(time.RFC3339)), 0644)
}

// Funkce performUpdate stáhne, rozbalí a zpracuje GTFS data, zaregistruje chybějící SD instance a uloží čas poslední aktualizace.
func performUpdate() {
	updateRunning = true
	defer func() { updateRunning = false }()

	log.Println("Starting GTFS update...")

	// Načti čas poslední aktualizace (pokud existuje).
	lastUpdate, err := readLastUpdateTime()
	if err == nil {
		if time.Since(lastUpdate) < maxAge {
			log.Println("GTFS data already updated recently. Skipping download.")
			return
		}
	} else {
		log.Println("No previous update timestamp found. Continuing...")
	}

	// Stáhni ZIP.
	if err := downloadGTFS(); err != nil {
		log.Println("Download failed:", err)
		return
	}

	// Rozbal ZIP do static_data.
	if err := unzipFile(localZipPath, staticDataDir); err != nil {
		log.Println("Unzip failed:", err)
		return
	}
	log.Println("GTFS data extracted successfully.")

	// Vygeneruj mapu trip key -> trip_id.
	err = GenerateTripKeyMap(staticDataDir)
	if err != nil {
		log.Println("Trip key map generation failed:", err)
		return
	}
	log.Println("trip_key_map.csv successfully updated.")

	// RabbitMQ client.
	client := rabbitmq.NewClient()
	log.Println("Successfully connected to RabbitMQ.")
	log.Println(client)
	defer client.Dispose()

	// Registruj nové SD instance.
	registerMissingSDInstances(client)

	// Ulož čas této aktualizace
	if err := saveLastUpdateTime(time.Now()); err != nil {
		log.Println("Failed to save update timestamp:", err)
	}
}

// Funkce downloadGTFS stáhne aktuální GTFS ZIP soubor z URL.
func downloadGTFS() error {
	resp, err := http.Get(gtfsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(localZipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Funkce unzipFile rozbalí stažený ZIP soubor do složky static_data.
func unzipFile(src, dest string) error {
	os.RemoveAll(dest)
	os.MkdirAll(dest, os.ModePerm)

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		dstFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		srcFile, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(dstFile, srcFile)
		dstFile.Close()
		srcFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
