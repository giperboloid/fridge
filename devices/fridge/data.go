package fridge

import (
	"time"
	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/devicems/models"
	"os"
	"github.com/giperboloid/devicems/config/fridgeconfig"
	"github.com/KharkivGophers/device-smart-house/tcp/connectionupdate"
)

//DataCollector gathers data from DataGenerator
//and sends completed request's structures to the ReqChan channel
func DataCollector(ticker *time.Ticker, cBot <-chan entities.FridgeGenerData, cTop <-chan entities.FridgeGenerData,
	ReqChan chan entities.FridgeRequest, stopInner chan struct{}) {
	var mTop = make(map[int64]float32)
	var mBot = make(map[int64]float32)

	for {
		select {
		case <-stopInner:
			log.Println("DataCollector(): wg.Done()")
			return
		case tv := <-cTop:
			mTop[tv.Time] = tv.Data
		case bv := <-cBot:
			mBot[bv.Time] = bv.Data
		case <-ticker.C:
			ReqChan <- constructReq(mTop, mBot)

			//Cleaning temp maps
			mTop = make(map[int64]float32)
			mBot = make(map[int64]float32)
		}
	}
}

//RunDataCollector setups DataCollector
func RunDataCollector(config *fridgeconfig.FridgeConfig, cBot <-chan entities.FridgeGenerData,
	cTop <-chan entities.FridgeGenerData, ReqChan chan entities.FridgeRequest, c *entities.Control) {

	duration := config.GetSendFreq()
	stopInner := make(chan struct{})
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)

	configChanged := make(chan struct{})
	config.AddSubIntoPool("DataCollector", configChanged)

	if config.GetTurned() {
		go DataCollector(ticker, cBot, cTop, ReqChan, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := config.GetTurned()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(config.GetSendFreq()) * time.Millisecond)
					go DataCollector(ticker, cBot, cTop, ReqChan, stopInner)
					log.Println("DataCollector() has been started")
				default:
					close(stopInner)
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(config.GetSendFreq()) * time.Millisecond)
					go DataCollector(ticker, cBot, cTop, ReqChan, stopInner)
					log.Println("DataCollector() has been started")
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(config.GetSendFreq()) * time.Millisecond)
				default:
					close(stopInner)
					log.Println("DataCollector() has been killed")
				}
			}
		case <- c.Controller:
			log.Error("Data Collector Failed")
			return
		}
	}
}

func constructReq(mTop map[int64]float32, mBot map[int64]float32) entities.FridgeRequest {
	var fridgeData entities.FridgeData
	args := os.Args[1:]

	fridgeData.TempCam2 = mBot
	fridgeData.TempCam1 = mTop

	req := entities.FridgeRequest{
		Action: "update",
		Time:   time.Now().UnixNano(),
		Meta: entities.Metadata{
			Type: args[0],
			Name: args[1],
			MAC:  args[2]},
		Data: fridgeData,
	}
	return req
}

//DataGenerator generates pseudo-random data that represents devices's behavior
func DataGenerator(ticker *time.Ticker, cBot chan<- entities.FridgeGenerData, cTop chan<- entities.FridgeGenerData,
	stopInner chan struct{}) {
	for {
		select {
		case <-ticker.C:
			cTop <- entities.FridgeGenerData{Time: makeTimestamp(), Data: rand.Float32() * 10}
			cBot <- entities.FridgeGenerData{Time: makeTimestamp(), Data: (rand.Float32() * 10) - 8}

		case <-stopInner:
			log.Println("DataGenerator(): wg.Done()")
			return
		}

	}
}

//RunDataGenerator setups DataGenerator
func RunDataGenerator(config *fridgeconfig.FridgeConfig, cBot chan<- entities.FridgeGenerData,
	cTop chan<- entities.FridgeGenerData, c *entities.Control) {

	duration := config.GetCollectFreq()
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)
	stopInner := make(chan struct{})

	configChanged := make(chan struct{})
	config.AddSubIntoPool("DataGenerator", configChanged)

	if config.GetTurned() {
		go DataGenerator(ticker, cBot, cTop, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := config.GetTurned()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(config.GetCollectFreq()) * time.Millisecond)
					go DataGenerator(ticker, cBot, cTop, stopInner)
					log.Println("DataGenerator() has been started")
				default:
					close(stopInner)
					ticker.Stop()
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(config.GetCollectFreq()) * time.Millisecond)
					go DataGenerator(ticker, cBot, cTop, stopInner)
					log.Println("DataGenerator() has been started")
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(config.GetCollectFreq()) * time.Millisecond)
				default:
					close(stopInner)
					log.Println("DataGenerator() has been killed")
				}
			}
		case <- c.Controller:
			log.Error("Data Generator Failed")
			return
		}
	}
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//DataTransfer func sends request as JSON to the centre
func DataTransfer(config *fridgeconfig.FridgeConfig, reqChan chan entities.FridgeRequest, c *entities.Control) {

	// for data transfer
	transferConnParams := entities.TransferConnParams{
		HostOut: connectionupdate.GetEnvCenter("CENTER_TCP_ADDR"),
		PortOut: "3030",
		ConnTypeOut: "tcp",
	}


	defer func() {
		if a := recover(); a != nil {
			log.Error(a)
			c.Close()
		}
	} ()
	conn := connectionupdate.GetDial(transferConnParams.ConnTypeOut, transferConnParams.HostOut, transferConnParams.PortOut)

	for {
		select {
		case r := <-reqChan:
			go func() {
				defer func() {
					if a := recover(); a != nil {
						log.Error(a)
						c.Close()
					}
				} ()
				connectionupdate.Send(r, conn)
			}()
		case <- c.Controller:
			log.Error("Data Transfer Failed")
			return
		}
	}
}
