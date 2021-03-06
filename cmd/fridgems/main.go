package main

import (
	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/kostiamol/fridgems/entities"
	"github.com/kostiamol/fridgems/services"
)

func main() {
	ctrl := &entities.ServiceController{
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
		retryInterval,
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
		retryInterval,
	)
	ds.Run()

	ctrl.Wait()
	logrus.Info("fridge is down")
}
