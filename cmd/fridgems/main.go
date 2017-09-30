package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/devicems/fridge"
	"github.com/giperboloid/devicems/entities"
)

func DefineDevice() []string {
	args := os.Args[1:]
	log.Warningln("Type:"+"["+args[0]+"];", "Name:"+"["+args[1]+"];", "MAC:"+"["+args[2]+"]")
	if len(args) < 3 {
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
		devType = device[0]
		ctrl    = &entities.RoutinesController{StopChan: make(chan struct{})}
	)

	configConn := entities.ConfigConn{
		ConnType: "tcp",
		Server: entities.Server{
			Host: GetEnvCentermsHost("CENTER_TCP_ADDR"),
			Port: "3000"},
	}

	if devType != "fridge" {
		log.Error("Unknown Device!")
		ctrl.Terminate()
	}

	fridge.Run(configConn.ConnType, configConn.Server, ctrl, device)

	ctrl.Wait()
	log.Info("fridgems: microservice is down")
}
