package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/fridge"
)

func main() {
	log.Infof("fridge: name:[%s] MAC:[%s]", devMeta.Name, devMeta.MAC)

	ctrl := &entities.RoutinesController{StopChan: make(chan struct{})}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("main(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	collectFridgeData := entities.CollectFridgeData{
		CTop:    make(chan entities.FridgeGenerData, 100),
		CBot:    make(chan entities.FridgeGenerData, 100),
		ReqChan: make(chan entities.FridgeRequest),
	}

	conf := fridge.NewConfiguration()
	conf.SetInitConfig(entities.Server{centermsHost, centermsDevConfigPort}, &devMeta, ctrl)

	go fridge.DataGenerator(conf, collectFridgeData.CBot, collectFridgeData.CTop, ctrl)
	go fridge.DataCollector(conf, collectFridgeData.CBot, collectFridgeData.CTop, collectFridgeData.ReqChan, ctrl)
	go fridge.DataSender(entities.Server{centermsHost, centermsDevDataPort}, collectFridgeData.ReqChan, ctrl)

	ctrl.Wait()
	log.Info("fridgems is down")
}
