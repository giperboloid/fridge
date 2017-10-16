package services

import (
	"bytes"
	"math/rand"
	"os"
	"time"

	"context"

	"encoding/json"

	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"google.golang.org/grpc"
	"github.com/Sirupsen/logrus"
)

type FridgeData struct {
	TempCam1 map[int64]float32
	TempCam2 map[int64]float32
}

type FridgeRequest struct {
	Time int64
	Meta entities.DevMeta
	Data FridgeData
}

type FridgeGenData struct {
	Time int64
	Data float32
}

type DataService struct {
	Config     *Configuration
	Controller *entities.ServicesController
	topCompart chan FridgeGenData
	botCompart chan FridgeGenData
	ReqChan    chan FridgeRequest
	Centerms   entities.Server
	Log        *logrus.Logger
}

func NewDataService(c *Configuration, s entities.Server, ctrl *entities.ServicesController, l *logrus.Logger) *DataService {
	return &DataService{
		topCompart: make(chan FridgeGenData, 100),
		botCompart: make(chan FridgeGenData, 100),
		ReqChan:    make(chan FridgeRequest),
		Config:     c,
		Centerms:   s,
		Controller: ctrl,
		Log:        l,
	}
}

func (s *DataService) Run() {
	go s.generateData()
	go s.collectData()
	go s.sendData()
}

func (s *DataService) generateData() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("generateData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	duration := s.Config.GetCollectFreq()
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)
	stopInner := make(chan struct{})

	configChanged := make(chan struct{})
	s.Config.Subscribe("dataGenerator", configChanged)

	if s.Config.GetTurnedOn() {
		go dataGenerator(ticker, s.botCompart, s.topCompart, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := s.Config.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetCollectFreq()) * time.Millisecond)
					go dataGenerator(ticker, s.botCompart, s.topCompart, stopInner)
					s.Log.Println("dataGenerator() is running")
				default:
					close(stopInner)
					ticker.Stop()
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetCollectFreq()) * time.Millisecond)
					go dataGenerator(ticker, s.botCompart, s.topCompart, stopInner)
					s.Log.Println("dataGenerator() is running")
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(s.Config.GetCollectFreq()) * time.Millisecond)
				default:
					close(stopInner)
					s.Log.Println("dataGenerator() is down")
				}
			}
		case <-s.Controller.StopChan:
			s.Log.Error("generateData() is down")
			return
		}
	}
}

func dataGenerator(t *time.Ticker, cBot chan<- FridgeGenData, cTop chan<- FridgeGenData,
	stopInner chan struct{}) {
	for {
		select {
		case <-t.C:
			cTop <- FridgeGenData{Time: currentTimestamp(), Data: rand.Float32() * 10}
			cBot <- FridgeGenData{Time: currentTimestamp(), Data: (rand.Float32() * 10) - 8}
		case <-stopInner:
			logrus.Info("dataGenerator() is down")
			return
		}
	}
}

func currentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (s *DataService) collectData() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("collectData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	duration := s.Config.GetSendFreq()
	stopInner := make(chan struct{})
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)

	configChanged := make(chan struct{})
	s.Config.Subscribe("dataCollector", configChanged)

	if s.Config.GetTurnedOn() {
		go dataCollector(ticker, s.botCompart, s.topCompart, s.ReqChan, stopInner)
	}

	for {
		select {
		case <-configChanged:
			state := s.Config.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
					go dataCollector(ticker, s.botCompart, s.topCompart, s.ReqChan, stopInner)
					s.Log.Println("dataCollector() is running")
				default:
					close(stopInner)
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
					go dataCollector(ticker, s.botCompart, s.topCompart, s.ReqChan, stopInner)
					s.Log.Println("dataCollector() is running")
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
				default:
					close(stopInner)
					s.Log.Println("dataCollector() is down")
				}
			}
		case <-s.Controller.StopChan:
			s.Log.Error("collectData() is down")
			return
		}
	}
}

func dataCollector(t *time.Ticker, cBot <-chan FridgeGenData, cTop <-chan FridgeGenData,
	ReqChan chan FridgeRequest, stopInner chan struct{}) {
	var mTop = make(map[int64]float32)
	var mBot = make(map[int64]float32)

	for {
		select {
		case <-stopInner:
			logrus.Println("dataCollector() is down")
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

func constructReq(mTop map[int64]float32, mBot map[int64]float32) FridgeRequest {
	var fd FridgeData
	args := os.Args[1:]

	fd.TempCam1 = mTop
	fd.TempCam2 = mBot

	fr := FridgeRequest{
		Time: time.Now().UnixNano(),
		Meta: entities.DevMeta{
			Type: "fridge",
			Name: args[0],
			MAC:  args[1],
		},
		Data: fd,
	}
	return fr
}

func (s *DataService) sendData() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("sendData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	conn := dial(s.Centerms)
	defer conn.Close()

	for {
		select {
		case r := <-s.ReqChan:
			go func() {
				defer func() {
					if r := recover(); r != nil {
						s.Log.Error(r)
						s.Controller.Terminate()
					}
				}()
				saveDevData(r, conn)
			}()
		case <-s.Controller.StopChan:
			s.Log.Error("sendData(): data sending has failed")
			return
		}
	}
}

func dial(s entities.Server) *grpc.ClientConn {
	var count int
	conn, err := grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
	for err != nil {
		if count >= 5 {
			panic("dial(): can't connect to the remote server")
		}
		time.Sleep(time.Second)
		conn, err = grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
		if err != nil {
			logrus.Printf("dial(): grpc.Dial has failed: %s", err)
		}
		count++
		logrus.Printf("dial(): reconnect count: %d", count)
	}
	return conn
}

func saveDevData(fr FridgeRequest, conn *grpc.ClientConn) {
	fr.Time = time.Now().UnixNano()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(fr.Data); err != nil {
		panic("FridgeData can't be encoded for sending")
	}

	pbfr := &pb.SaveDevDataRequest{
		Time: fr.Time,
		Meta: &pb.DevMeta{
			Type: fr.Meta.Type,
			Name: fr.Meta.Name,
			Mac:  fr.Meta.MAC,
		},
		Data: buf.Bytes(),
	}

	client := pb.NewCenterServiceClient(conn)
	resp, err := client.SaveDevData(context.Background(), pbfr)
	if err != nil {
		logrus.Println("saveFridgeData(): SaveFridgeData() has failed", err)
	}
	logrus.Printf("centerms has received DevData with status: %s", resp.Status)
}
