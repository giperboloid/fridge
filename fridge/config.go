package fridge

import (
	"encoding/json"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
)

const (
	devType = "fridge"
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
		log.Error("RequestConfig(): can't connect to the centerms: " + s.Host + ":" + s.Port)
		panic("centerms hasn't been found")
	}

	req := entities.FridgeRequest{
		Action: "config",
		Meta: entities.DevMeta{
			Type: devType,
			Name: args[0],
			MAC:  args[1],
		},
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		log.Errorf("askConfig(): Encode JSON: %s", err)
	}

	var rfc entities.FridgeConfig
	if err := json.NewDecoder(conn).Decode(&rfc); err != nil {
		log.Errorf("askConfig(): Decode JSON: %s", err)
	}

	log.Infof("current config: %+v", rfc)

	if err != nil && rfc.IsEmpty() {
		panic("connection has been closed by the centerms")
	}

	c.update(&rfc)

	go func() {
		for {
			defer func() {
				if r := recover(); r != nil {
					ctrl.Terminate()
				}
			}()
			c.listen(conn)
		}
	}()
}

func (c *Configuration) update(nfc *entities.FridgeConfig) {
	c.TurnedOn = nfc.TurnedOn
	c.SendFreq = nfc.SendFreq
	c.CollectFreq = nfc.CollectFreq
	if c.TurnedOn {
		log.Info("fridge is turned on")
	} else {
		log.Info("fridge is turned off")
	}
}

func (c *Configuration) listen(conn net.Conn) {
	var fc *entities.FridgeConfig
	if err := json.NewDecoder(conn).Decode(&fc); err != nil {
		panic("centerms hasn't been found")
	}

	c.update(fc)
	go c.publish()

	r := entities.Response{
		Descr: "FridgeConfig has been received",
	}

	c.update(fc)
	go c.publish()

	if err := json.NewEncoder(conn).Encode(&r); err != nil {
		log.Errorf("listen(): Encode JSON: %s", err)
	}
}

func (c *Configuration) publish() {
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