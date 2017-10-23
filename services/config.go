package services

import (
	"encoding/json"
	"sync"

	"bytes"
	"encoding/binary"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"golang.org/x/net/context"
)

type FridgeConfig struct {
	TurnedOn    bool
	CollectFreq int64
	SendFreq    int64
}

type Configuration struct {
	sync.Mutex
	FridgeConfig
	SubsPool map[string]chan struct{}
}

func (c *Configuration) Subscribe(key string, value chan struct{}) {
	c.Mutex.Lock()
	c.SubsPool[key] = value
	c.Mutex.Unlock()
}

func (c *Configuration) GetTurnedOn() bool {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.TurnedOn
}
func (c *Configuration) SetTurnedOn(b bool) {
	c.Mutex.Lock()
	c.TurnedOn = b
	c.Mutex.Unlock()
}

func (c *Configuration) GetCollectFreq() int64 {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.CollectFreq
}
func (c *Configuration) SetCollectFreq(cf int64) {
	c.Mutex.Lock()
	c.CollectFreq = cf
	c.Mutex.Unlock()
}

func (c *Configuration) GetSendFreq() int64 {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.SendFreq
}
func (c *Configuration) SetSendFreq(sf int64) {
	c.Mutex.Lock()
	c.SendFreq = sf
	c.Mutex.Unlock()
}

type ConfigService struct {
	Config     *Configuration
	Center     entities.Server
	Controller *entities.ServicesController
	Meta       *entities.DevMeta
	Log        *logrus.Logger
}

func NewConfigService(m *entities.DevMeta, s entities.Server, ctrl *entities.ServicesController,
	l *logrus.Logger) *ConfigService {
	return &ConfigService{
		Meta: m,
		Config: &Configuration{
			SubsPool: make(map[string]chan struct{}),
		},
		Center:     s,
		Controller: ctrl,
		Log:        l,
	}
}

func (s *ConfigService) SetInitConfig() {
	pbic := &pb.SetDevInitConfigRequest{
		Time: time.Now().UnixNano(),
		Meta: &pb.DevMeta{
			Type: s.Meta.Type,
			Name: s.Meta.Name,
			Mac:  s.Meta.MAC,
		},
	}

	conn := dialCenter(s.Center)
	defer conn.Close()

	client := pb.NewCenterServiceClient(conn)
	resp, err := client.SetDevInitConfig(context.Background(), pbic)
	if err != nil {
		s.Log.Error("SetInitConfig(): SetDevInitConfig() has failed: ", err)
		panic("init config hasn't been received")
	}

	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, resp.Config); err != nil {
		s.Log.Error("SetInitConfig(): Write() has failed: ", err)
		panic("init config translation to []byte has failed")
	}

	s.updateConfig(buf)
}

func (s *ConfigService) ListenDevConfig() {
	conn, _ := nats.Connect(nats.DefaultURL)
	s.Log.Infof("Connected to " + nats.DefaultURL)

	queue := "Config.ConfigPatchQueue"
	subject := "Config.Patch." + s.Meta.MAC

	conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		eventStore := pb.EventStore{}
		if err := proto.Unmarshal(msg.Data, &eventStore); err == nil {
			s.updateConfig(bytes.NewBufferString(eventStore.EventData))
		}
	})
}

func (s *ConfigService) updateConfig(buf *bytes.Buffer) {
	var temp = s.Config.FridgeConfig
	if err := json.NewDecoder(buf).Decode(&temp); err != nil {
		s.Log.Error("updateConfig(): Decode() has failed: ", err)
		panic("config decoding has failed")
	}

	if temp.TurnedOn && !s.Config.TurnedOn {
		s.Log.Info("fridge is running")
	} else if !temp.TurnedOn && s.Config.TurnedOn {
		s.Log.Info("fridge is on pause")
	}

	s.Config.FridgeConfig = temp
	s.Log.Infof("current config: %+v", s.Config.FridgeConfig)
	s.Config.publishConfigIsPatched()
}

func (c *Configuration) publishConfigIsPatched() {
	for _, v := range c.SubsPool {
		v <- struct{}{}
	}
}
