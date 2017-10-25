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
	"google.golang.org/grpc/connectivity"
)

type FridgeData struct {
	TopCompart map[int64]float32
	BotCompart map[int64]float32
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
	Config         *Configuration
	Meta           *entities.DevMeta
	Controller     *entities.ServicesController
	TopCompart     chan FridgeGenData
	BotCompart     chan FridgeGenData
	ReqChan        chan SaveFridgeDataRequest
	Center         entities.Server
	Log            *logrus.Logger
	ReconnInterval time.Duration
}

func NewDataService(c *Configuration, m *entities.DevMeta, s entities.Server, ctrl *entities.ServicesController,
	l *logrus.Logger, reconn time.Duration) *DataService {
	return &DataService{
		TopCompart:     make(chan FridgeGenData, 100),
		BotCompart:     make(chan FridgeGenData, 100),
		ReqChan:        make(chan SaveFridgeDataRequest),
		Config:         c,
		Meta:           m,
		Center:         s,
		Controller:     ctrl,
		Log:            l,
		ReconnInterval: reconn,
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
			s.Log.Errorf("DataService: generateData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	duration := s.Config.GetCollectFreq()
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)
	stopInner := make(chan struct{})

	configIsPatched := make(chan struct{})
	s.Config.Subscribe("dataGenerator", configIsPatched)

	if s.Config.GetTurnedOn() {
		go s.dataGenerator(ticker, s.TopCompart, s.BotCompart, stopInner)
	}

	for {
		select {
		case <-configIsPatched:
			state := s.Config.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetCollectFreq()) * time.Millisecond)
					go s.dataGenerator(ticker, s.TopCompart, s.BotCompart, stopInner)
				default:
					close(stopInner)
					ticker.Stop()
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetCollectFreq()) * time.Millisecond)
					go s.dataGenerator(ticker, s.TopCompart, s.BotCompart, stopInner)
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
			s.Log.Info("data generation has stopped")
			return
		}
	}
}

func (s *DataService) dataGenerator(t *time.Ticker, topCompart chan<- FridgeGenData, botCompart chan<- FridgeGenData,
	stopInner chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: dataGenerator(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	for {
		select {
		case <-t.C:
			topCompart <- FridgeGenData{Time: currentTimestamp(), Data: rand.Float32() * 10}
			botCompart <- FridgeGenData{Time: currentTimestamp(), Data: (rand.Float32() * 10) - 8}
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
			s.Log.Errorf("DataService: collectData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	duration := s.Config.GetSendFreq()
	stopInner := make(chan struct{})
	ticker := time.NewTicker(time.Duration(duration) * time.Millisecond)

	configIsPatched := make(chan struct{})
	s.Config.Subscribe("dataCollection", configIsPatched)

	if s.Config.GetTurnedOn() {
		go s.dataCollector(ticker, s.TopCompart, s.BotCompart, s.ReqChan, stopInner)
	}

	for {
		select {
		case <-configIsPatched:
			state := s.Config.GetTurnedOn()
			switch state {
			case true:
				select {
				case <-stopInner:
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
					go s.dataCollector(ticker, s.TopCompart, s.BotCompart, s.ReqChan, stopInner)
				default:
					close(stopInner)
					stopInner = make(chan struct{})
					ticker = time.NewTicker(time.Duration(s.Config.GetSendFreq()) * time.Millisecond)
					go s.dataCollector(ticker, s.TopCompart, s.BotCompart, s.ReqChan, stopInner)
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
			s.Log.Info("data collection has stopped")
			return
		}
	}
}

func (s *DataService) dataCollector(t *time.Ticker, topCompart <-chan FridgeGenData, botCompart <-chan FridgeGenData,
	ReqChan chan SaveFridgeDataRequest, stopInner chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: dataCollector(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	var timeTempTopCompart = make(map[int64]float32)
	var timeTempBotCompart = make(map[int64]float32)

	for {
		select {
		case t := <-topCompart:
			timeTempTopCompart[t.Time] = t.Data
		case b := <-botCompart:
			timeTempBotCompart[b.Time] = b.Data
		case <-t.C:
			ReqChan <- s.newSaveFridgeDataRequest(timeTempTopCompart, timeTempBotCompart)
			timeTempTopCompart = make(map[int64]float32)
			timeTempBotCompart = make(map[int64]float32)
		case <-stopInner:
			return
		}
	}
}

func (s *DataService) newSaveFridgeDataRequest(topCompart map[int64]float32, botCompart map[int64]float32) SaveFridgeDataRequest {
	return SaveFridgeDataRequest{
		Time: time.Now().UnixNano(),
		Meta: *s.Meta,
		Data: FridgeData{
			TopCompart: topCompart,
			BotCompart: botCompart,
		},
	}
}

func (s *DataService) sendData() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("DataService: sendData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	conn := dial(s.Center, s.Log, s.ReconnInterval)
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
			s.Log.Errorf("DataService: saveFridgeData(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	fr.Time = time.Now().UnixNano()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(fr.Data); err != nil {
		s.Log.Errorf("DataService: saveFridgeData(): Encode() has failed: %s", err)
		panic("FridgeData can't be encoded for sending")
	}

	req := &pb.SaveDevDataRequest{
		Time: fr.Time,
		Meta: &pb.DevMeta{
			Type: fr.Meta.Type,
			Name: fr.Meta.Name,
			Mac:  fr.Meta.MAC,
		},
		Data: buf.Bytes(),
	}

	client := pb.NewCenterServiceClient(conn)
	for conn.GetState() != connectivity.Ready {
		s.Log.Error("DataService: saveFridgeData(): center connectivity status: NOT READY")
		duration := time.Duration(rand.Intn(int(s.ReconnInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
	}

	resp, err := client.SaveDevData(context.Background(), req)
	if err != nil {
		s.Log.Errorf("DataService: saveFridgeData(): SaveDevData() has failed: %s", err)
	}
	s.Log.Infof("center has received FridgeData with status: %s", resp.Status)
}

func dial(s entities.Server, l *logrus.Logger, reconnInterval time.Duration) *grpc.ClientConn {
	conn, err := grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
	for err != nil {
		l.Error("dial(): grpc.Dial(): failed to dial remote server")
		duration := time.Duration(rand.Intn(int(reconnInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		conn, err = grpc.Dial(s.Host+":"+s.Port, grpc.WithInsecure())
	}
	return conn
}
