package shared
package shared

import (
	"sync"
	"time"
)

type DroneCommand struct {
	OccurrenceID string
	Action       string
	Timestamp    time.Time
}

type MissionResult struct {
	OccurrenceID string
	Action       string
	Status       string
}

type Incident struct {
	OccurrenceID string

}