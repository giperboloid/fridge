package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/services"
)

func main() {
	logrus.Infof("device type: [%s] name:[%s] MAC:[%s]", fridgeMeta.Type, fridgeMeta.Name, fridgeMeta.MAC)

	ctrl := &entities.ServicesController{
		StopChan: make(chan struct{}),
	}

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("main(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	cs := services.NewConfigService(
		&fridgeMeta,
		entities.Server{
			Host: centerHost,
			Port: centerConfigPort,
		},
		ctrl,
		logrus.New(),
		reconnInterval,
	)
	cs.Run()

	ds := services.NewDataService(
		cs.Config,
		&fridgeMeta,
		entities.Server{
			Host: centerHost,
			Port: centerDataPort,
		},
		ctrl,
		logrus.New(),
		reconnInterval,
	)
	ds.Run()

	ctrl.Wait()
	logrus.Info("fridge is down")
}
