// Package services provides two services for device configuration
// and data handling.
package services

import (
	"encoding/json"
	"math/rand"
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
	"google.golang.org/grpc/connectivity"
)

// FridgeConfig is used to store fridge configuration.
// TurnedOn    specifies whether device is on or off.
// CollectFreq specifies frequency for data collection on device.
// SendFreq    specifies frequency for sending data from device to center.
type FridgeConfig struct {
	TurnedOn    bool
	CollectFreq int64
	SendFreq    int64
}

// Configuration is used to store fridge's configuration and to
// handle its subscribers.
type Configuration struct {
	sync.RWMutex
	FridgeConfig
	SubsPool map[string]chan struct{}
}

// Subscribe subscribes clients to configuration patches.
func (c *Configuration) Subscribe(subscriber string, ch chan struct{}) {
	c.RWMutex.Lock()
	c.SubsPool[subscriber] = ch
	c.RWMutex.Unlock()
}

func (c *Configuration) GetTurnedOn() bool {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.TurnedOn
}
func (c *Configuration) SetTurnedOn(turnedOn bool) {
	c.RWMutex.Lock()
	c.TurnedOn = turnedOn
	c.RWMutex.Unlock()
}

func (c *Configuration) GetCollectFreq() int64 {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.CollectFreq
}
func (c *Configuration) SetCollectFreq(collectFreq int64) {
	c.RWMutex.Lock()
	c.CollectFreq = collectFreq
	c.RWMutex.Unlock()
}

func (c *Configuration) GetSendFreq() int64 {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.SendFreq
}
func (c *Configuration) SetSendFreq(sendFreq int64) {
	c.RWMutex.Lock()
	c.SendFreq = sendFreq
	c.RWMutex.Unlock()
}

// ConfigService is used to handle device's configuration parameters
// manipulation.
type ConfigService struct {
	Config         *Configuration
	Center         entities.Server
	Controller     *entities.ServicesController
	Meta           *entities.DevMeta
	Log            *logrus.Logger
	ReconnInterval time.Duration
}

// NewConfigService creates and initializes new ConfigService object.
// It returns initialized object.
func NewConfigService(m *entities.DevMeta, s entities.Server, ctrl *entities.ServicesController,
	l *logrus.Logger, reconn time.Duration) *ConfigService {
	return &ConfigService{
		Meta: m,
		Config: &Configuration{
			SubsPool: make(map[string]chan struct{}),
		},
		Center:         s,
		Controller:     ctrl,
		Log:            l,
		ReconnInterval: reconn,
	}
}

// Run sets initial device configuration and listens
// for configuration patches from the center.
func (s *ConfigService) Run() {
	s.setInitConfig()
	go s.listenConfigPatch()
}

func (s *ConfigService) setInitConfig() {
	req := &pb.SetDevInitConfigRequest{
		Time: time.Now().UnixNano(),
		Meta: &pb.DevMeta{
			Type: s.Meta.Type,
			Name: s.Meta.Name,
			Mac:  s.Meta.MAC,
		},
	}

	conn := dial(s.Center, s.Log, s.ReconnInterval)
	defer conn.Close()

	client := pb.NewCenterServiceClient(conn)
	for conn.GetState() != connectivity.Ready {
		s.Log.Error("ConfigService: setInitConfig(): center connectivity status: NOT READY")
		duration := time.Duration(rand.Intn(int(s.ReconnInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
	}

	resp, err := client.SetDevInitConfig(context.Background(), req)
	if err != nil {
		s.Log.Error("ConfigService: setInitConfig(): SetDevInitConfig() has failed: ", err)
		panic("init config hasn't been received")
	}

	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, resp.Config); err != nil {
		s.Log.Error("ConfigService: setInitConfig(): Write() has failed: ", err)
		panic("init config translation to []byte has failed")
	}

	s.updateConfig(buf)
}

func (s *ConfigService) listenConfigPatch() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: listenConfigPatch(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	conn, err := nats.Connect(nats.DefaultURL)
	for err != nil {
		s.Log.Error("ConfigService: listenConfigPatch(): nats connectivity status: DISCONNECTED")
		duration := time.Duration(rand.Intn(int(s.ReconnInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		conn, err = nats.Connect(nats.DefaultURL)
	}

	s.Log.Infof("connected to " + nats.DefaultURL)

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
		s.Log.Error("ConfigService: updateConfig(): Decode() has failed: ", err)
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
