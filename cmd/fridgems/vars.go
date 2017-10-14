package main

import (
	"os"

	"github.com/giperboloid/fridgems/entities"
)

var (
	devMeta = entities.DevMeta{
		Type: "fridge",
		Name: getDevName(),
		MAC:  getDevMAC(),
	}

	fridgemsHost = getEnvServerHost("FRIDGEMS_TCP_ADDR")
	fridgemsConfigPort = "4000"

	centermsHost       = getEnvServerHost("CENTERMS_TCP_ADDR")
	centermsDataPort   = "3030"
	centermsConfigPort = "3000"
)

func getDevName() string {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("device name is missing")
	}
	return args[0]
}

func getDevMAC() string {
	args := os.Args[1:]
	if len(args) < 2 {
		panic("device mac is missing")
	}
	return args[1]
}

func getEnvServerHost(key string) string {
	host := os.Getenv(key)
	if len(host) == 0 {
		return "127.0.0.1"
	}
	return host
}
