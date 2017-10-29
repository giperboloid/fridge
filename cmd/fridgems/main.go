package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/services"
	"flag"
)

func main() {
	ctrl := &entities.ServicesController{
		StopChan: make(chan struct{}),
	}

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("main(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	flag.StringVar(&devMeta.Name, "name", "", "device name")
	flag.StringVar(&devMeta.MAC, "mac", "", "device MAC")
	flag.Parse()
	checkCLIArgs()

	logrus.Infof("device type: [%s] name:[%s] MAC:[%s]", devMeta.Type, devMeta.Name, devMeta.MAC)

	cs := services.NewConfigService(
		&devMeta,
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
		&devMeta,
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
