package main

import (
	"os"

	"github.com/giperboloid/fridgems/entities"
)

var (
	devType  = "fridge"
	devName  = getDevName()
	devMAC   = getDevMAC()

	connType = "tcp"
	centerms = entities.Server{
		Host: getEnvCentermsHost("CENTERMS_TCP_ADDR"),
		Port: getEnvCentermsPort("CENTERMS_TCP_PORT"),
	}
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

func getEnvCentermsHost(key string) string {
	host := os.Getenv(key)
	if len(host) == 0 {
		return "127.0.0.1"
	}
	return host
}

func getEnvCentermsPort(key string) string {
	port := os.Getenv(key)
	if len(port) == 0 {
		return "3000"
	}
	return port
}
