package main

import (
	"os"

	"time"

	"github.com/giperboloid/fridgems/entities"
)

const (
	localhost               = "127.0.0.1"
	defaultCenterConfigPort = "3092"
	defaultCenterDataPort   = "3126"
	reconnInterval          = time.Second * 10
)

var (
	fridgeMeta = entities.DevMeta{
		Type: "fridge",
		Name: getDevName(),
		MAC:  getDevMAC(),
	}
	centerHost       = getEnvVar("CENTER_TCP_ADDR", localhost)
	centerDataPort   = getEnvVar("CENTER_DATA_TCP_PORT", defaultCenterDataPort)
	centerConfigPort = getEnvVar("CENTER_CONFIG_TCP_PORT", defaultCenterConfigPort)
)

// getDevName checks whether device name was passed as an argument for the app.
// It returns that name if it was passed and panics otherwise.
func getDevName() string {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("device name is missing")
	}
	return args[0]
}

// getDevMAC checks whether device MAC was passed as an argument for the app.
// It returns that MAC if it was passed and panics otherwise.
func getDevMAC() string {
	args := os.Args[1:]
	if len(args) < 2 {
		panic("device mac is missing")
	}
	return args[1]
}

// getEnvVar checks whether environmental variable with name 'key' was specified.
// It returns that variable if it was set and defaultVal otherwise.
func getEnvVar(key string, defaultVal string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		return defaultVal
	}
	return val
}
