package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/fridge"
)

func main() {
	var (
		fridgeParams = GetFridgeNameAndMAC()
		ctrl         = &entities.RoutinesController{StopChan: make(chan struct{})}
	)

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("fridgems: main(): panic: %s", r)
			ctrl.Terminate()
		}
	}()

	log.Infof("fridge: name: [%s] MAC: [%s]", fridgeParams[0], fridgeParams[1])

	collectFridgeData := entities.CollectFridgeData{
		CTop:    make(chan entities.FridgeGenerData, 100),
		CBot:    make(chan entities.FridgeGenerData, 100),
		ReqChan: make(chan entities.FridgeRequest),
	}

	c := fridge.NewConfiguration()
	c.RequestConfig(connType, &centerms, ctrl, fridgeParams)

	go fridge.DataGenerator(c, collectFridgeData.CBot, collectFridgeData.CTop, ctrl)
	go fridge.DataCollector(c, collectFridgeData.CBot, collectFridgeData.CTop, collectFridgeData.ReqChan, ctrl)
	go fridge.DataSender(centerms, collectFridgeData.ReqChan, ctrl)

	ctrl.Wait()
	log.Info("fridgems: microservice is down")
}
