package fridge

import (
	"encoding/json"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
)

type FridgeConfig struct {
	sync.Mutex
	TurnedOn    bool
	CollectFreq int64
	SendFreq    int64
	SubsPool    map[string]chan struct{}
}

func Run(connType string, s entities.Server, rc *entities.RoutinesController, args []string) {
	collectFridgeData := entities.CollectFridgeData{
		CTop:    make(chan entities.FridgeGenerData, 100), // First Camera
		CBot:    make(chan entities.FridgeGenerData, 100), // Second Camera
		ReqChan: make(chan entities.FridgeRequest),
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Run config has failed!: %s", r)
		}
	}()

	fc := NewFridgeConfig()
	fc.requestConfig(connType, s, rc, args)

	go RunDataGenerator(fc, collectFridgeData.CBot, collectFridgeData.CTop, rc)
	go RunDataCollector(fc, collectFridgeData.CBot, collectFridgeData.CTop, collectFridgeData.ReqChan, rc)
	go DataTransfer(s, collectFridgeData.ReqChan, rc)
}

func NewFridgeConfig() *FridgeConfig {
	conf := &FridgeConfig{}
	conf.SubsPool = make(map[string]chan struct{})
	return conf
}

func (fc *FridgeConfig) GetTurnedOn() bool {
	fc.Mutex.Lock()
	defer fc.Mutex.Unlock()
	return fc.TurnedOn
}
func (fc *FridgeConfig) SetTurned(b bool) {
	fc.Mutex.Lock()
	fc.TurnedOn = b
	fc.Mutex.Unlock()
}

func (fc *FridgeConfig) GetCollectFreq() int64 {
	fc.Mutex.Lock()
	defer fc.Mutex.Unlock()
	return fc.CollectFreq
}
func (fc *FridgeConfig) SetCollectFreq(cf int64) {
	fc.Mutex.Lock()
	fc.CollectFreq = cf
	fc.Mutex.Unlock()
}

func (fc *FridgeConfig) GetSendFreq() int64 {
	fc.Mutex.Lock()
	defer fc.Mutex.Unlock()
	return fc.SendFreq
}
func (fc *FridgeConfig) SetSendFreq(sf int64) {
	fc.Mutex.Lock()
	fc.SendFreq = sf
	fc.Mutex.Unlock()
}

func (fc *FridgeConfig) AddSubscriber(key string, value chan struct{}) {
	fc.Mutex.Lock()
	fc.SubsPool[key] = value
	fc.Mutex.Unlock()
}
func (fc *FridgeConfig) RemoveSubscriber(key string) {
	fc.Mutex.Lock()
	delete(fc.SubsPool, key)
	fc.Mutex.Unlock()
}

func listenConfig(devConfig *FridgeConfig, conn net.Conn) {
	var fc entities.FridgeConfig

	if err := json.NewDecoder(conn).Decode(&fc); err != nil {
		panic("No config found!")
	}

	r := entities.Response{
		Descr: "Config have been received",
	}
	devConfig.update(fc)
	go publishConfig(devConfig)

	if err := json.NewEncoder(conn).Encode(&r); err != nil {
		log.Errorf("listenConfig(): Encode JSON: %s", err)
	}
}

func publishConfig(d *FridgeConfig) {
	for _, v := range d.SubsPool {
		v <- struct{}{}
	}
}

func (fc *FridgeConfig) update(nfc entities.FridgeConfig) {
	fc.TurnedOn = nfc.TurnedOn
	fc.SendFreq = nfc.SendFreq
	log.Warningln("SendFreq: ", fc.SendFreq)
	fc.CollectFreq = nfc.CollectFreq
	log.Warningln("CollectFreq: ", fc.CollectFreq)

	switch fc.TurnedOn {
	case false:
		log.Info("Status: ON PAUSE")
	case true:
		log.Info("Status: RUNNING")
	}
}

func (fc *FridgeConfig) requestConfig(connType string,s entities.Server, c *entities.RoutinesController, args []string) {
	conn, err := net.Dial(connType, s.Host+":"+s.Port)
	for err != nil {
		log.Error("Can't connect to the server: " + s.Host + ":" + s.Port)
		panic("No center found!")
	}

	req := entities.FridgeRequest{
		Action: "config",
		Meta: entities.DevMeta{
			Type: "fridge",
			Name: args[0],
			MAC:  args[1],
		},
	}

	log.Info("FridgeRequest:", req)

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		log.Errorf("askConfig(): Encode JSON: %s", err)
	}

	log.Info("FridgeRequest:", req)

	var rfc entities.FridgeConfig
	if err := json.NewDecoder(conn).Decode(&rfc); err != nil {
		log.Errorf("askConfig(): Decode JSON: %s", err)
	}

	if err != nil && rfc.IsEmpty() {
		panic("Connection has been closed by center")
	}

	fc.update(rfc)

	go func() {
		for {
			defer func() {
				if r := recover(); r != nil {
					c.Terminate()
					log.Error("Initialization Failed")
				}
			}()
			listenConfig(fc, conn)
		}
	}()
}
