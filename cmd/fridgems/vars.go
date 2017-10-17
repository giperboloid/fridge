package main

import (
	"os"

	"github.com/giperboloid/fridgems/entities"
)

var (
	fridgeMeta = entities.DevMeta{
		Type: "fridge",
		Name: getDevName(),
		MAC:  getDevMAC(),
	}

	fridgeHost       = getEnvServerHost("FRIDGE_TCP_ADDR")
	fridgeConfigPort = "4040"

	centerHost       = getEnvServerHost("CENTER_TCP_ADDR")
	centerDataPort   = getEnvCenterDataPort("CENTER_DATA_PORT")
	centerConfigPort = getEnvCenterConfigPort("CENTER_CONFIG_PORT")
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

func getEnvCenterDataPort(key string) string {
	host := os.Getenv(key)
	if len(host) == 0 {
		return "3030"
	}
	return host
}

func getEnvCenterConfigPort(key string) string {
	host := os.Getenv(key)
	if len(host) == 0 {
		return "3000"
	}
	return host
}