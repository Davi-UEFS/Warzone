package main

import (
	"sync/atomic"
)

var INCIDENT_MESSAGES = map[string]string{
	"incendio": "Incêndio detectado!",
	"oleo":     "Vazamento de óleo detectado!",
}

var commandCounter atomic.Uint64
