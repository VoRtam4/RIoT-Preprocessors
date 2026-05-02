package main

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/rabbitmq"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedModel"
	"github.com/MichalBures-OG/bp-bures-RIoT-commons/src/sharedUtils"
)

const (
	fetchDelay      = 2 * time.Minute
	startupJitter   = 500 * time.Millisecond
	wazeSDTypeUID   = "WAZE_JAM_LOCATION"
	wazeSDTypeLabel = "Waze Jam Location"
	unknownTagValue = "unknown"
)

var (
	sdInstances           = sharedUtils.NewSet[sharedModel.SDInstanceInfo]()
	sdInstancesMutex      sync.Mutex
	activeDevices         = make(map[string]deviceSnapshot)
	activeDevicesMu       sync.Mutex
	preprocessorStartedAt = time.Now().UTC()
)

type wazeFeed struct {
	Jams []map[string]interface{} `json:"jams"`
}

type lineCoordinate struct {
	X float64
	Y float64
}

type segmentReference struct {
	ID        int64
	FromNode  int64
	ToNode    int64
	IsForward bool
}

type deviceSnapshot struct {
	Label string
	Tags  map[string]string
}

type deviceAggregate struct {
	UID             string
	Label           string
	EventTime       time.Time
	Tags            map[string]string
	RawJams         []map[string]interface{}
	JamCount        int
	Delay           float64
	Length          float64
	Level           float64
	Speed           float64
	SpeedKPH        float64
	PubMillisLatest int64
}

func main() {
	log.SetOutput(os.Stderr)

	config := loadConfig()
	client := rabbitmq.NewClient()
	defer client.Dispose()

	registerSDType(client)
	time.Sleep(5 * time.Second)
	go checkForSetOfSDInstancesUpdates(client)

	for {
		fetchAndProcessWazeData(client, config)
		time.Sleep(fetchDelay)
	}
}
