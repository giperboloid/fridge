package main

import (
	"os"
	"github.com/giperboloid/fridgems/entities"
)

var (
	connType = "tcp"
	centerms = entities.Server{
		Host: GetEnvCentermsHost("CENTERMS_TCP_ADDR"),
		Port: "3000",
	}
)

func GetFridgeNameAndMAC() []string {
	args := os.Args[1:]
	if len(args) < 2 {
		panic("number of parameters for fridge initialization is less than is needed")
	}
	return args
}

func GetEnvCentermsHost(key string) string {
	host := os.Getenv(key)
	if len(host) == 0 {
		return "127.0.0.1"
	}
	return host
}