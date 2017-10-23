package services

import (
	"bytes"
	"math/rand"
	"time"

	"context"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"google.golang.org/grpc"
)

type FridgeData struct {
	TempTopCompart map[int64]float32
	TempBotCompart map[int64]float32
}

type SaveFridgeDataRequest struct {
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
	Meta       *entities.DevMeta
	Controller *entities.ServicesController
	topCompart chan FridgeGenData
	botCompart chan FridgeGenData
	ReqChan    chan SaveFridgeDataRequest
	Center     entities.Server
	Log        *logrus.Logger
}

func NewDataService(c *Configuration, m *entities.DevMeta, s entities.Server, ctrl *entities.ServicesController, l *logrus.Logger) *DataService {
	return &DataService{
		topCompart: make(chan FridgeGenData, 100),
		botCompart: make(chan FridgeGenData, 100),
		ReqChan:    make(chan SaveFridgeDataRequest),
		Config:     c,
		Meta:       m,
		Center:     s,
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
		go s.dataGenerator(ticker, s.botCompart, s.topCompart, stopInner)
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
					go s.dataGenerator(ticker, s.botCompart, s.topCompart, stopInner)
				default:
					close(stopInner)
					ticker.Stop()
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetCollectFreq()) * time.Millisecond)
					go s.dataGenerator(ticker, s.botCompart, s.topCompart, stopInner)
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(s.Config.GetCollectFreq()) * time.Millisecond)
				default:
					close(stopInner)
				}
			}
		case <-s.Controller.StopChan:
			s.Log.Error("data generation has stopped")
			return
		}
	}
}

func (s *DataService) dataGenerator(t *time.Ticker, cBot chan<- FridgeGenData, cTop chan<- FridgeGenData,
	stopInner chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("dataGenerator(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	for {
		select {
		case <-t.C:
			cTop <- FridgeGenData{Time: currentTimestamp(), Data: rand.Float32() * 10}
			cBot <- FridgeGenData{Time: currentTimestamp(), Data: (rand.Float32() * 10) - 8}
		case <-stopInner:
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

	configIsChanged := make(chan struct{})
	s.Config.Subscribe("dataCollection", configIsChanged)

	if s.Config.GetTurnedOn() {
		go s.dataCollector(ticker, s.botCompart, s.topCompart, s.ReqChan, stopInner)
	}

	for {
		select {
		case <-configIsChanged:
			state := s.Config.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
					go s.dataCollector(ticker, s.botCompart, s.topCompart, s.ReqChan, stopInner)
				default:
					close(stopInner)
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
					go s.dataCollector(ticker, s.botCompart, s.topCompart, s.ReqChan, stopInner)
				}
			case false:
				select {
				case <-stopInner:
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
				default:
					close(stopInner)
				}
			}
		case <-s.Controller.StopChan:
			s.Log.Error("data collection has stopped")
			return
		}
	}
}

func (s *DataService) dataCollector(t *time.Ticker, cBot <-chan FridgeGenData, cTop <-chan FridgeGenData,
	ReqChan chan SaveFridgeDataRequest, stopInner chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("dataCollector(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	var mTop = make(map[int64]float32)
	var mBot = make(map[int64]float32)

	for {
		select {
		case tv := <-cTop:
			mTop[tv.Time] = tv.Data
		case bv := <-cBot:
			mBot[bv.Time] = bv.Data
		case <-t.C:
			ReqChan <- s.createFridgeRequest(mTop, mBot)
			mTop = make(map[int64]float32)
			mBot = make(map[int64]float32)
		case <-stopInner:
			return
		}
	}
}

func (s *DataService) createFridgeRequest(tempTopCompart map[int64]float32, tempBotCompart map[int64]float32) SaveFridgeDataRequest {
	return SaveFridgeDataRequest{
		Time: time.Now().UnixNano(),
		Meta: *s.Meta,
		Data: FridgeData{
			TempTopCompart: tempTopCompart,
			TempBotCompart: tempBotCompart,
		},
	}
}

func (s *DataService) sendData() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("sendData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	conn := dialCenter(s.Center)
	defer conn.Close()

	for {
		select {
		case r := <-s.ReqChan:
			go s.saveFridgeData(r, conn)
		case <-s.Controller.StopChan:
			s.Log.Info("data sending has stopped")
			return
		}
	}
}

func (s *DataService) saveFridgeData(fr SaveFridgeDataRequest, conn *grpc.ClientConn) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("saveFridgeData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	fr.Time = time.Now().UnixNano()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(fr.Data); err != nil {
		s.Log.Errorf("saveFridgeData(): Encode() has failed: %s", err)
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
		s.Log.Errorf("saveFridgeData(): SaveDevData() has failed: %s", err)
	}
	s.Log.Infof("center has received FridgeData with status: %s", resp.Status)
}

func dialCenter(s entities.Server) *grpc.ClientConn {
	var count int
	conn, err := grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
	for err != nil {
		if count >= 5 {
			panic("dialCenter(): can't connect to the remote server")
		}
		time.Sleep(time.Second)
		conn, err = grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
		if err != nil {
			logrus.Printf("dialCenter(): grpc.Dial has failed: %s", err)
		}
		count++
		logrus.Printf("dialCenter(): reconnect count: %d", count)
	}
	return conn
}
