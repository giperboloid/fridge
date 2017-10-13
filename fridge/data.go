package fridge

import (
	"bytes"
	"math/rand"
	"os"
	"time"

	"context"

	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"google.golang.org/grpc"
)

//DataCollector setups dataCollector
func DataCollector(c *Configuration, cBot <-chan entities.FridgeGenerData,
	cTop <-chan entities.FridgeGenerData, ReqChan chan entities.FridgeRequest, ctrl *entities.RoutinesController) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("DataCollector(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	duration := c.GetSendFreq()
	stopInner := make(chan struct{})
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)

	configChanged := make(chan struct{})
	c.Subscribe("dataCollector", configChanged)

	if c.GetTurnedOn() {
		go dataCollector(ticker, cBot, cTop, ReqChan, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := c.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(c.GetSendFreq()) * time.Millisecond)
					go dataCollector(ticker, cBot, cTop, ReqChan, stopInner)
					log.Println("dataCollector() has been started")
				default:
					close(stopInner)
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(c.GetSendFreq()) * time.Millisecond)
					go dataCollector(ticker, cBot, cTop, ReqChan, stopInner)
					log.Println("dataCollector() has been started")
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(c.GetSendFreq()) * time.Millisecond)
				default:
					close(stopInner)
					log.Println("dataCollector() has been killed")
				}
			}
		case <-ctrl.StopChan:
			log.Error("DataCollector() is down")
			return
		}
	}
}

//dataCollector gathers data from dataGenerator
//and sends completed request's structures to the ReqChan channel
func dataCollector(t *time.Ticker, cBot <-chan entities.FridgeGenerData, cTop <-chan entities.FridgeGenerData,
	ReqChan chan entities.FridgeRequest, stopInner chan struct{}) {
	var mTop = make(map[int64]float32)
	var mBot = make(map[int64]float32)

	for {
		select {
		case <-stopInner:
			log.Println("dataCollector(): wg.Done()")
			return
		case tv := <-cTop:
			mTop[tv.Time] = tv.Data
		case bv := <-cBot:
			mBot[bv.Time] = bv.Data
		case <-t.C:
			ReqChan <- constructReq(mTop, mBot)
			//Cleaning temp maps
			mTop = make(map[int64]float32)
			mBot = make(map[int64]float32)
		}
	}
}

func constructReq(mTop map[int64]float32, mBot map[int64]float32) entities.FridgeRequest {
	var fd entities.FridgeData
	args := os.Args[1:]

	fd.TempCam2 = mBot
	fd.TempCam1 = mTop

	fr := entities.FridgeRequest{
		Time:   time.Now().UnixNano(),
		Meta: entities.DevMeta{
			Type: "fridge",
			Name: args[0],
			MAC:  args[1],
		},
		Data: fd,
	}
	return fr
}

//DataGenerator setups dataGenerator
func DataGenerator(c *Configuration, cBot chan<- entities.FridgeGenerData,
	cTop chan<- entities.FridgeGenerData, ctrl *entities.RoutinesController) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("DataGenerator(): panic(): %s", r)
			ctrl.Terminate()
		}
	}()

	duration := c.GetCollectFreq()
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)
	stopInner := make(chan struct{})

	configChanged := make(chan struct{})
	c.Subscribe("dataGenerator", configChanged)

	if c.GetTurnedOn() {
		go dataGenerator(ticker, cBot, cTop, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := c.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(c.GetCollectFreq()) * time.Millisecond)
					go dataGenerator(ticker, cBot, cTop, stopInner)
					log.Println("dataGenerator() has been started")
				default:
					close(stopInner)
					ticker.Stop()
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(c.GetCollectFreq()) * time.Millisecond)
					go dataGenerator(ticker, cBot, cTop, stopInner)
					log.Println("dataGenerator() has been started")
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(c.GetCollectFreq()) * time.Millisecond)
				default:
					close(stopInner)
					log.Println("dataGenerator() has been killed")
				}
			}
		case <-ctrl.StopChan:
			log.Error("DataGenerator() is down")
			return
		}
	}
}

//dataGenerator generates pseudo-random data that represents devices's behavior
func dataGenerator(t *time.Ticker, cBot chan<- entities.FridgeGenerData, cTop chan<- entities.FridgeGenerData,
	stopInner chan struct{}) {
	for {
		select {
		case <-t.C:
			cTop <- entities.FridgeGenerData{Time: currentTimestamp(), Data: rand.Float32() * 10}
			cBot <- entities.FridgeGenerData{Time: currentTimestamp(), Data: (rand.Float32() * 10) - 8}
		case <-stopInner:
			log.Println("dataGenerator(): wg.Done()")
			return
		}
	}
}

func currentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//DataSender func sends request as JSON to the centre
func DataSender(s entities.Server, reqChan chan entities.FridgeRequest, c *entities.RoutinesController) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("DataSender(): panic(): %s", r)
			c.Terminate()
		}
	}()

	conn := dial(s)
	defer conn.Close()

	for {
		select {
		case r := <-reqChan:
			go func() {
				defer func() {
					if a := recover(); a != nil {
						log.Error(a)
						c.Terminate()
					}
				}()
				send(r, conn)
			}()
		case <-c.StopChan:
			log.Error("DataSender(): data sending has failed")
			return
		}
	}
}

func dial(s entities.Server) *grpc.ClientConn {
	var count int
	conn, err := grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
	for err != nil {
		if count >= 5 {
			panic("dial(): can't connect to the centerms")
		}
		time.Sleep(time.Second)
		conn, err = grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
		if err != nil {
			log.Errorf("getDial(): %s", err)
		}
		count++
		log.Infof("reconnect count: %d", count)
	}
	return conn
}

func send(fr entities.FridgeRequest, conn *grpc.ClientConn) {
	fr.Time = time.Now().UnixNano()
	client := pb.NewCenterServiceClient(conn)

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(fr.Data)
	if err != nil {
		panic("FridgeData can't be encoded for sending")
	}

	pbfr := &pb.SaveDevDataRequest{
		Time:   fr.Time,
		Meta: &pb.DevMeta{
			Type: fr.Meta.Type,
			Name: fr.Meta.Name,
			Mac:  fr.Meta.MAC,
		},
		Data: buf.Bytes(),
	}

	saveDevData(client, pbfr)
}

func saveDevData(c pb.CenterServiceClient, req *pb.SaveDevDataRequest) {
	resp, err := c.SaveDevData(context.Background(), req)
	if err != nil {
		log.Error("saveFridgeData(): SaveFridgeData() has failed", err)
	}
	log.Infof("centerms has received data with status: %s", resp.Status)
}
