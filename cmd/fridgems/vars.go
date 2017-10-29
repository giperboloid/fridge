package main

import (
	"os"

	"time"

	"github.com/giperboloid/fridgems/entities"
)

const (
	devType                 = "fridge"
	localhost               = "127.0.0.1"
	defaultCenterConfigPort = "3092"
	defaultCenterDataPort   = "3126"
	reconnInterval          = time.Second * 10
)

var (
	devMeta = entities.DevMeta{
		Type: devType,
	}
	centerHost       = getEnvVar("CENTER_TCP_ADDR", localhost)
	centerDataPort   = getEnvVar("CENTER_DATA_TCP_PORT", defaultCenterDataPort)
	centerConfigPort = getEnvVar("CENTER_CONFIG_TCP_PORT", defaultCenterConfigPort)
)

// getEnvVar checks whether environmental variable with name 'key' was specified.
// It returns that variable if it was set and defaultVal otherwise.
func getEnvVar(key string, defaultVal string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		return defaultVal
	}
	return val
}

// checkCLIArgs checks whether vital args were passed. If not - panic occurs.
func checkCLIArgs() {
	if len(devMeta.Name) == 0 {
		panic("device name is missing")
	}

	if len(devMeta.MAC) == 0 {
		panic("device MAC is missing")
	}
}
