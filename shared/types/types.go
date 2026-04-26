package types

import (
	"time"
)

type DroneCommand struct {
	AtomicID  int
	Action    string
	Timestamp time.Time
}
