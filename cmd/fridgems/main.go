package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/services"
	"github.com/logrus"
	"github.com/giperboloid/fridgems/api"
	"time"
)

func main() {
	log.Infof("fridgems is running on fridge with name:[%s] and MAC:[%s]", devMeta.Name, devMeta.MAC)

	ctrl := &entities.ServicesController{StopChan: make(chan struct{})}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("main(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	cs := services.NewConfigService(&devMeta, entities.Server{centermsHost, centermsConfigPort}, ctrl, logrus.New())
	cs.SetInitConfig()
	config := cs.GetConfig()

	reconnect := time.NewTicker(time.Second * 3)
	a := api.NewAPI(cs, reconnect, entities.Server{fridgemsHost,fridgemsConfigPort})
	go a.Listen()

	ds := services.NewDataService(config, entities.Server{centermsHost, centermsDataPort}, ctrl, logrus.New())
	ds.Run()

	ctrl.Wait()
	log.Info("fridgems is down")
}
