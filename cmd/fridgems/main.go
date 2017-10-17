package main

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/services"
	"github.com/giperboloid/fridgems/api/grpcsvc"
)

func main() {
	logrus.Infof("fridge is running with name:[%s] and MAC:[%s]", fridgeMeta.Name, fridgeMeta.MAC)

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
		&fridgeMeta,
		entities.Server{
			Host: centerHost,
			Port: centerConfigPort,
		},
		ctrl,
		logrus.New(),
	)

	cs.SetInitConfig()
	config := cs.Config

	grpcsvc.Init(grpcsvc.GRPCConfig{
		ConfigService: cs,
		Reconnect: time.NewTicker(time.Second * 3),
		Server: entities.Server{
			Host: fridgeHost,
			Port: fridgeConfigPort,
		},
	})

	ds := services.NewDataService(
		config,
		&fridgeMeta,
		entities.Server{
			Host: centerHost,
			Port: centerDataPort,
		},
		ctrl,
		logrus.New(),
	)
	ds.Run()

	ctrl.Wait()
	logrus.Info("fridge is down")
}
