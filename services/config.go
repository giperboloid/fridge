package services

import (
	"encoding/json"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"time"
	"golang.org/x/net/context"
	"encoding/binary"
	"bytes"
	"fmt"
	"google.golang.org/grpc"
)

type Configuration struct {
	sync.Mutex
	TurnedOn    bool
	CollectFreq int64
	SendFreq    int64
	SubsPool    map[string]chan struct{}
}

type ConfigService struct {
	Config     *Configuration
	Center     entities.Server
	Controller *entities.ServicesController
	DevMeta    *entities.DevMeta
}

func NewConfigService(m *entities.DevMeta, s entities.Server, ctrl *entities.ServicesController) *ConfigService {
	return &ConfigService {
		DevMeta: m,
		Config: &Configuration{
			SubsPool: make(map[string]chan struct{}),
		},
		Center: s,
		Controller: ctrl,
	}
}

func (s *ConfigService) SetInitConfig() {
	pbic := &pb.SetDevInitConfigRequest{
		Time:   time.Now().UnixNano(),
		Meta: &pb.DevMeta{
			Type: s.DevMeta.Type,
			Name: s.DevMeta.Name,
			Mac:  s.DevMeta.MAC,
		},
	}

	conn := dial(s.Center)
	defer conn.Close()

	client := pb.NewCenterServiceClient(conn)

	r, err := client.SetDevInitConfig(context.Background(), pbic)
	if err != nil {
		log.Error("SetInitConfig(): SetDevInitConfig() has failed: ", err)
		panic("init config hasn't been received")
	}

	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, r.Config); err != nil {
		panic(err)
	}

	var fc entities.FridgeConfig
	if err := json.NewDecoder(buf).Decode(&fc); err != nil {
		panic("init config decoding has failed")
	}

	log.Infof("init config: %+v", fc)
	s.update(&fc)

	go s.listenConfigPatch(time.NewTicker(time.Second * 3))
}

func (s *ConfigService) GetConfig() *Configuration {
	return s.Config
}

func (s *ConfigService) listenConfigPatch(r *time.Ticker) {
	ln, err := net.Listen("tcp", "127.0.0.1"+":"+fmt.Sprint("4000"))
	if err != nil {
		log.Errorf("listenConfigPatch(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range r.C {
			ln, err = net.Listen("tcp", "127.0.0.1"+":"+fmt.Sprint("4000"))
			if err != nil {
				log.Errorf("listenConfigPatch(): Listen() has failed: %s", err)
			}
		}
		r.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterDevServiceServer(gs, s)
	gs.Serve(ln)
}

func (s *ConfigService) PatchDevConfig(ctx context.Context, r *pb.PatchDevConfigRequest) (*pb.PatchDevConfigResponse, error) {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, r.Config); err != nil {
		panic(err)
	}

	var fc entities.FridgeConfig
	if err := json.NewDecoder(buf).Decode(&fc); err != nil {
		panic("config patch decoding has failed")
	}

	log.Infof("config patch: %+v", fc)
	s.update(&fc)
	return &pb.PatchDevConfigResponse{Status:"OK"}, nil
}

func (s *ConfigService) update(nfc *entities.FridgeConfig) {
	if nfc.TurnedOn && !s.Config.TurnedOn {
		log.Info("services is turned on")
	} else if !nfc.TurnedOn && s.Config.TurnedOn {
		log.Info("services is turned off")
	}

	s.Config.TurnedOn = nfc.TurnedOn
	s.Config.CollectFreq = nfc.CollectFreq
	s.Config.SendFreq = nfc.SendFreq
}

func (s *Configuration) publishConfigPatch() {
	for _, v := range s.SubsPool {
		v <- struct{}{}
	}
}

func (s *Configuration) Subscribe(key string, value chan struct{}) {
	s.Mutex.Lock()
	s.SubsPool[key] = value
	s.Mutex.Unlock()
}
func (s *Configuration) Unsubscribe(key string) {
	s.Mutex.Lock()
	delete(s.SubsPool, key)
	s.Mutex.Unlock()
}

func (s *Configuration) GetTurnedOn() bool {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	return s.TurnedOn
}
func (s *Configuration) SetTurnedOn(b bool) {
	s.Mutex.Lock()
	s.TurnedOn = b
	s.Mutex.Unlock()
}

func (s *Configuration) GetCollectFreq() int64 {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	return s.CollectFreq
}
func (s *Configuration) SetCollectFreq(cf int64) {
	s.Mutex.Lock()
	s.CollectFreq = cf
	s.Mutex.Unlock()
}

func (s *Configuration) GetSendFreq() int64 {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	return s.SendFreq
}
func (s *Configuration) SetSendFreq(sf int64) {
	s.Mutex.Lock()
	s.SendFreq = sf
	s.Mutex.Unlock()
}
