package fridge

import (
	"math/rand"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/devicems/entities"
	"google.golang.org/grpc"
	"github.com/giperboloid/devicems/pb"
	"context"
)

//RunDataCollector setups DataCollector
func RunDataCollector(fc *FridgeConfig, cBot <-chan entities.FridgeGenerData,
	cTop <-chan entities.FridgeGenerData, ReqChan chan entities.FridgeRequest, c *entities.RoutinesController) {

	duration := fc.GetSendFreq()
	stopInner := make(chan struct{})
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)

	configChanged := make(chan struct{})
	fc.AddSubscriber("DataCollector", configChanged)

	if fc.GetTurnedOn() {
		go DataCollector(ticker, cBot, cTop, ReqChan, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := fc.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(fc.GetSendFreq()) * time.Millisecond)
					go DataCollector(ticker, cBot, cTop, ReqChan, stopInner)
					log.Println("DataCollector() has been started")
				default:
					close(stopInner)
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(fc.GetSendFreq()) * time.Millisecond)
					go DataCollector(ticker, cBot, cTop, ReqChan, stopInner)
					log.Println("DataCollector() has been started")
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(fc.GetSendFreq()) * time.Millisecond)
				default:
					close(stopInner)
					log.Println("DataCollector() has been killed")
				}
			}
		case <-c.StopChan:
			log.Error("Data Collector Failed")
			return
		}
	}
}

//DataCollector gathers data from DataGenerator
//and sends completed request's structures to the ReqChan channel
func DataCollector(t *time.Ticker, cBot <-chan entities.FridgeGenerData, cTop <-chan entities.FridgeGenerData,
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
		Action: "update",
		Time:   time.Now().UnixNano(),
		Meta: entities.DevMeta{
			Type: args[0],
			Name: args[1],
			MAC:  args[2]},
		Data: fd,
	}
	return fr
}

//RunDataGenerator setups DataGenerator
func RunDataGenerator(config *FridgeConfig, cBot chan<- entities.FridgeGenerData,
	cTop chan<- entities.FridgeGenerData, c *entities.RoutinesController) {

	duration := config.GetCollectFreq()
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)
	stopInner := make(chan struct{})

	configChanged := make(chan struct{})
	config.AddSubscriber("DataGenerator", configChanged)

	if config.GetTurnedOn() {
		go DataGenerator(ticker, cBot, cTop, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := config.GetTurnedOn()
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
		case <-c.StopChan:
			log.Error("Data Generator Failed")
			return
		}
	}
}

//DataGenerator generates pseudo-random data that represents devices's behavior
func DataGenerator(t *time.Ticker, cBot chan<- entities.FridgeGenerData, cTop chan<- entities.FridgeGenerData,
	stopInner chan struct{}) {
	for {
		select {
		case <-t.C:
			cTop <- entities.FridgeGenerData{Time: makeTimestamp(), Data: rand.Float32() * 10}
			cBot <- entities.FridgeGenerData{Time: makeTimestamp(), Data: (rand.Float32() * 10) - 8}
		case <-stopInner:
			log.Println("DataGenerator(): wg.Done()")
			return
		}
	}
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//DataTransfer func sends request as JSON to the centre
func DataTransfer(s entities.Server, reqChan chan entities.FridgeRequest, c *entities.RoutinesController) {
	transferConn := entities.TransferConn{
		Server: entities.Server{
			Host: s.Host,
			Port: "3030"},
		ConnType: "tcp",
	}

	defer func() {
		if a := recover(); a != nil {
			log.Error(a)
			c.Terminate()
		}
	}()

	conn := Dial(transferConn.Server)
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
				Send(r, conn)
			}()
		case <-c.StopChan:
			log.Error("Data Transfer Failed")
			return
		}
	}
}

func Dial(s entities.Server) *grpc.ClientConn {
	var count int
	conn, err := grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
	for err != nil {
		if count >= 5 {
			panic("Can't connect to the server: send")
		}
		time.Sleep(time.Second)
		conn, err = grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
		if err != nil {
			log.Errorf("getDial(): %s", err)
		}
		count++
		log.Warningln("Reconnect count: ", count)
	}
	return conn
}

func Send(fr entities.FridgeRequest, conn *grpc.ClientConn) {
	fr.Time = time.Now().UnixNano()
	client := pb.NewFridgeServiceClient(conn)

	pbfr := &pb.FridgeDataRequest{
		Action: fr.Action,
		Time: fr.Time,
		Meta: &pb.DevMeta{
			Type: fr.Meta.Type,
			Name: fr.Meta.Name,
			Mac: fr.Meta.MAC,
		},
		Data: &pb.FridgeData{
			TempCam1: fr.Data.TempCam1,
			TempCam2: fr.Data.TempCam2,
		},
	}

	setFridgeData(client, pbfr)

	log.Infoln("Data was sent. Response from center: ")
}

func setFridgeData(c pb.FridgeServiceClient, req *pb.FridgeDataRequest) {
	resp, err := c.SetFridgeData(context.Background(), req)
	if err != nil {
		log.Fatalf("Could not create FridgeData: %v", err)
	}
	if resp.Status {
		log.Printf("Data has been received with status: %s", resp.Status)
	}
}
