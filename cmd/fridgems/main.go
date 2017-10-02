package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/fridge"
	"github.com/giperboloid/fridgems/entities"
)

func DefineDevice() []string {
	args := os.Args[1:]
	log.Warningln("Name:"+"["+args[0]+"];", "MAC:"+"["+args[1]+"]")
	if len(args) < 2 {
		panic("Incorrect devices's information")
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

func main() {
	var (
		device  = DefineDevice()
		ctrl    = &entities.RoutinesController{StopChan: make(chan struct{})}
	)

	configConn := entities.ConfigConn{
		ConnType: "tcp",
		Server: entities.Server{
			Host: GetEnvCentermsHost("CENTER_TCP_ADDR"),
			Port: "3000"},
	}

	fridge.Run(configConn.ConnType, configConn.Server, ctrl, device)

	ctrl.Wait()
	log.Info("fridgems: microservice is down")
}
