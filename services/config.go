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
	"github.com/giperboloid/fridgems/api/pb"
	"github.com/giperboloid/fridgems/entities"
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

// PublishConfigIsPatched notifies all the subscribers about
// new configuration patch.
func (c *Configuration) publishConfigIsPatched() {
	c.RWMutex.RLock()
	for _, v := range c.SubsPool {
		v <- struct{}{}
	}
	c.RWMutex.RUnlock()
}

// GetFridgeConfig returns value of FridgeConfig field.
func (c *Configuration) GetFridgeConfig() FridgeConfig {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.FridgeConfig
}

// SetFridgeConfig sets value for FridgeConfig field.
func (c *Configuration) SetFridgeConfig(fc FridgeConfig) {
	c.RWMutex.Lock()
	c.FridgeConfig = fc
	c.RWMutex.Unlock()
}

// GetTurnedOn returns value of TurnedOn field.
func (c *Configuration) GetTurnedOn() bool {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.TurnedOn
}

// SetTurnedOn sets value for TurnedOn field.
func (c *Configuration) SetTurnedOn(turnedOn bool) {
	c.RWMutex.Lock()
	c.TurnedOn = turnedOn
	c.RWMutex.Unlock()
}

// GetCollectFreq returns value of CollectFreq field.
func (c *Configuration) GetCollectFreq() int64 {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.CollectFreq
}

// SetCollectFreq sets value for CollectFreq field.
func (c *Configuration) SetCollectFreq(collectFreq int64) {
	c.RWMutex.Lock()
	c.CollectFreq = collectFreq
	c.RWMutex.Unlock()
}

// GetSendFreq returns value of SendFreq field.
func (c *Configuration) GetSendFreq() int64 {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.SendFreq
}

// SetSendFreq sets value for SendFreq field.
func (c *Configuration) SetSendFreq(sendFreq int64) {
	c.RWMutex.Lock()
	c.SendFreq = sendFreq
	c.RWMutex.Unlock()
}

// ConfigService is used to handle device's configuration parameters
// manipulation.
type ConfigService struct {
	Config        *Configuration
	Center        entities.Server
	Controller    *entities.ServiceController
	Meta          *entities.DevMeta
	Log           *logrus.Logger
	RetryInterval time.Duration
}

// NewConfigService creates and initializes new ConfigService object.
// It returns initialized object.
func NewConfigService(m *entities.DevMeta, s entities.Server, ctrl *entities.ServiceController,
	l *logrus.Logger, r time.Duration) *ConfigService {
	return &ConfigService{
		Meta: m,
		Config: &Configuration{
			SubsPool: make(map[string]chan struct{}),
		},
		Center:        s,
		Controller:    ctrl,
		Log:           l,
		RetryInterval: r,
	}
}

// Run sets initial device configuration and listens
// for configuration patches from the center.
func (s *ConfigService) Run() {
	s.setInitConfig()
	go s.listenConfigPatches()
}

func (s *ConfigService) setInitConfig() {
	req := &api.SetDevInitConfigRequest{
		Time: time.Now().UnixNano(),
		Meta: &api.DevMeta{
			Type: s.Meta.Type,
			Name: s.Meta.Name,
			Mac:  s.Meta.MAC,
		},
	}

	conn := dial(s.Center, s.Log, s.RetryInterval)
	defer conn.Close()

	client := api.NewCenterServiceClient(conn)
	for conn.GetState() != connectivity.Ready {
		s.Log.Error("ConfigService: setInitConfig(): center connectivity status: NOT READY")
		duration := time.Duration(rand.Intn(int(s.RetryInterval.Seconds())))
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

	s.patchConfig(buf)
}

func (s *ConfigService) listenConfigPatches() {
	defer func() {
		if r := recover(); r != nil {
			s.Log.Errorf("ConfigService: listenConfigPatches(): panic(): %s", r)
			s.Controller.Terminate()
		}
	}()

	conn, err := nats.Connect(nats.DefaultURL)
	for err != nil {
		s.Log.Error("ConfigService: listenConfigPatches(): nats connectivity status: DISCONNECTED")
		duration := time.Duration(rand.Intn(int(s.RetryInterval.Seconds())))
		time.Sleep(time.Second*duration + 1)
		conn, err = nats.Connect(nats.DefaultURL)
	}

	s.Log.Infof("connected to " + nats.DefaultURL)

	queue := "Config.ConfigPatchQueue"
	subject := "Config.Patch." + s.Meta.MAC

	conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		eventStore := api.EventStore{}
		if err := proto.Unmarshal(msg.Data, &eventStore); err == nil {
			s.patchConfig(bytes.NewBufferString(eventStore.EventData))
		}
	})
}

func (s *ConfigService) patchConfig(buf *bytes.Buffer) {
	var patchedConfig = s.Config.GetFridgeConfig()
	if err := json.NewDecoder(buf).Decode(&patchedConfig); err != nil {
		s.Log.Error("ConfigService: patchConfig(): Decode() has failed: ", err)
		panic("config decoding has failed")
	}

	if patchedConfig.TurnedOn && !s.Config.GetTurnedOn() {
		s.Log.Info("fridge is running")
	} else if !patchedConfig.TurnedOn && s.Config.GetTurnedOn() {
		s.Log.Info("fridge is on pause")
	}

	s.Config.SetFridgeConfig(patchedConfig)
	s.Log.Infof("current config: %+v", s.Config.FridgeConfig)
	s.Config.publishConfigIsPatched()
}
