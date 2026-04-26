package main

import (
	"fmt"
	"os"
)

var MISSION_TOPIC = fmt.Sprintf("drones/requisitions/%s", os.Getenv("CLIENT_ID"))
