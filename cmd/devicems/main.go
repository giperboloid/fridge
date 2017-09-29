package main

import (
	"github.com/giperboloid/devicems/config"
	"github.com/giperboloid/devicems/models"
	"github.com/giperboloid/devicems/tcp"
	log "github.com/Sirupsen/logrus"
)

func main() {
	configConnParams := entities.ConfigConnParams{
		ConnTypeConf: "tcp",
		HostConf:     tcp.GetEnvCenter("CENTER_TCP_ADDR"),
		PortConf:     "3000",
	}

	newDevice := config.CreateDevice()
	newDeviceType := newDevice[0]
	control := &entities.Control{
		Controller: make(chan struct{}),
	}

	switch newDeviceType {
	case "washer":
		startWasher(configConnParams.ConnTypeConf, configConnParams.HostConf, configConnParams.PortConf, control, newDevice)
	case "fridge":
		startFridge(configConnParams.ConnTypeConf, configConnParams.HostConf, configConnParams.PortConf, control, newDevice)
	default:
		log.Error("Unknown Device!")
		control.Close()
	}

	control.Wait()
	log.Info("Device has been terminated due to the center's issue")
}