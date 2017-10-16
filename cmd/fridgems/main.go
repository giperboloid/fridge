package main

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/services"
	"github.com/giperboloid/fridgems/api/grpcsvc"
)

func main() {
	logrus.Infof("fridge is running with name:[%s] and MAC:[%s]", devMeta.Name, devMeta.MAC)

	ctrl := &entities.ServicesController{
		StopChan: make(chan struct{},
	)}

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("main(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	cs := services.NewConfigService(
		&devMeta,
		entities.Server{
			Host: centermsHost,
			Port: centermsConfigPort,
		},
		ctrl,
		logrus.New(),
	)

	cs.SetInitConfig()
	config := cs.GetConfig()

	grpcsvc.Init(grpcsvc.GRPCConfig{
		ConfigService: cs,
		Reconnect: time.NewTicker(time.Second * 3),
		Server: entities.Server{
			Host: fridgemsHost,
			Port: fridgemsConfigPort,
		},
	})

	ds := services.NewDataService(
		config,
		entities.Server{
			Host: centermsHost,
			Port: centermsDataPort,
		},
		ctrl,
		logrus.New(),
	)
	ds.Run()

	ctrl.Wait()
	logrus.Info("fridge is down")
}
