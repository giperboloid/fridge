package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/services"
)

func main() {
	log.Infof("services: name:[%s] MAC:[%s]", devMeta.Name, devMeta.MAC)

	ctrl := &entities.ServicesController{StopChan: make(chan struct{})}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("main(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	cs := services.NewConfigService(&devMeta, entities.Server{centerHost, centerDevConfigPort}, ctrl)
	cs.SetInitConfig()
	config := cs.GetConfig()

	ds := services.NewDataService(config, entities.Server{centerHost, centerDevDataPort}, ctrl)
	ds.Run()

	ctrl.Wait()
	log.Info("fridgems is down")
}
