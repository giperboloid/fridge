package fridge

import (
	"encoding/json"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
)

type Configuration struct {
	sync.Mutex
	TurnedOn    bool
	CollectFreq int64
	SendFreq    int64
	SubsPool    map[string]chan struct{}
}

func NewConfiguration() *Configuration {
	c := &Configuration{}
	c.SubsPool = make(map[string]chan struct{})
	return c
}

func (c *Configuration) RequestConfig(connType string, s *entities.Server, ctrl *entities.RoutinesController, args []string) {
	conn, err := net.Dial(connType, s.Host+":"+s.Port)
	for err != nil {
		log.Error("RequestConfig: can't connect to the centerms: " + s.Host+":"+s.Port)
		panic("centerms hasn't been found")
	}

	req := entities.FridgeRequest{
		Action: "config",
		Meta: entities.DevMeta{
			Type: args[0],
			Name: args[1],
			MAC:  args[2]},
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		log.Errorf("askConfig(): Encode JSON: %s", err)
	}

	log.Info("Request:", req)

	var fc *entities.FridgeConfig
	if err := json.NewDecoder(conn).Decode(fc); err != nil {
		log.Errorf("askConfig(): Decode JSON: %s", err)
	}

	if err != nil && fc.IsEmpty() {
		panic("connection has been closed by centerms")
	}

	c.updateConfig(fc)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("RequestConfig(): listenConfig() has failed")
				ctrl.Terminate()
			}
		}()
		for {
			c.listenConfig(conn)
		}
	}()
}

func (c *Configuration) updateConfig(nfc *entities.FridgeConfig) {
	c.TurnedOn = nfc.TurnedOn
	c.SendFreq = nfc.SendFreq
	c.CollectFreq = nfc.CollectFreq
	if c.TurnedOn {
		log.Info("fridgems is turned on")
	} else {
		log.Info("fridgems is turned off")
	}
}

func (c *Configuration) listenConfig(conn net.Conn) {
	var fc *entities.FridgeConfig
	if err := json.NewDecoder(conn).Decode(fc); err != nil {
		panic("No config found!")
	}

	c.updateConfig(fc)
	go c.publishConfig()

	r := entities.Response{
		Descr: "FridgeConfig has been received",
	}
	if err := json.NewEncoder(conn).Encode(&r); err != nil {
		log.Errorf("listenConfig(): Encode JSON: %s", err)
	}
}

func (c *Configuration) publishConfig() {
	for _, v := range c.SubsPool {
		v <- struct{}{}
	}
}

func (c *Configuration) Subscribe(key string, value chan struct{}) {
	c.Mutex.Lock()
	c.SubsPool[key] = value
	c.Mutex.Unlock()
}
func (c *Configuration) Unsubscribe(key string) {
	c.Mutex.Lock()
	delete(c.SubsPool, key)
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