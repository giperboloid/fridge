package config

import (
	"github.com/giperboloid/devicems/entities"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"sync"
	"net"
)

type FridgeConfig struct {
	sync.Mutex
	turned      bool
	collectFreq int64
	sendFreq    int64
	subsPool    map[string]chan struct{}
}

func NewFridgeConfig() *FridgeConfig {
	conf := &FridgeConfig{}
	conf.subsPool = make(map[string]chan struct{})
	return conf
}

func (dfc *FridgeConfig) GetTurned() bool {
	dfc.Mutex.Lock()
	defer dfc.Mutex.Unlock()
	return dfc.turned
}
func (dfc *FridgeConfig) SetTurned(b bool) {
	dfc.Mutex.Lock()
	dfc.turned = b
	dfc.Mutex.Unlock()
}

func (dfc *FridgeConfig) GetCollectFreq() int64 {
	dfc.Mutex.Lock()
	defer dfc.Mutex.Unlock()
	return dfc.collectFreq
}
func (dfc *FridgeConfig) SetCollectFreq(b int64) {
	dfc.Mutex.Lock()
	dfc.collectFreq = b
	dfc.Mutex.Unlock()

}

func (dfc *FridgeConfig) GetSendFreq() int64 {
	dfc.Mutex.Lock()
	defer dfc.Mutex.Unlock()
	return dfc.sendFreq
}
func (dfc *FridgeConfig) SetSendFreq(b int64) {
	dfc.Mutex.Lock()
	dfc.sendFreq = b
	dfc.Mutex.Unlock()
}

func (dfc *FridgeConfig) AddSubIntoPool(key string, value chan struct{}) {
	dfc.Mutex.Lock()
	dfc.subsPool[key] = value
	dfc.Mutex.Unlock()
}
func (dfc *FridgeConfig) RemoveSubFromPool(key string) {
	dfc.Mutex.Lock()
	delete(dfc.subsPool, key)
	dfc.Mutex.Unlock()
}

func listenConfig(devConfig *FridgeConfig, conn net.Conn) {
	var resp entities.Response
	var conf entities.FridgeConfig

	if err := json.NewDecoder(conn).Decode(&conf); err != nil {
		panic("No config found!")
	}

	resp.Descr = "Config have been received"
	devConfig.updateConfig(conf)
	go publishConfig(devConfig)

	if err := json.NewEncoder(conn).Encode(&resp); err != nil {
		log.Errorf("listenConfig(): Encode JSON: %s", err)
	}
}

func publishConfig(d *FridgeConfig) {
	for _, v := range d.subsPool {
		v <- struct{}{}
	}
}

func (dfc *FridgeConfig) updateConfig(c entities.FridgeConfig) {
	dfc.turned = c.TurnedOn
	dfc.sendFreq = c.SendFreq
	log.Warningln("SendFreq: ", dfc.sendFreq)
	dfc.collectFreq = c.CollectFreq
	log.Warningln("CollectFreq: ", dfc.collectFreq)

	switch dfc.turned {
	case false:
		log.Warningln("ON PAUSE")
	case true:
		log.Warningln("WORKING")
	}
}

func (dfc *FridgeConfig) RequestFridgeConfig(connType string, host string, port string, c *entities.Control, args []string) {

	conn, err := net.Dial(connType, host+":"+port)
	for err != nil {
		log.Error("Can't connect to the server: " + host + ":" + port)
		panic("No center found!")
	}

	var response entities.FridgeConfig
	var request entities.FridgeRequest

	request = entities.FridgeRequest{
		Action: "config",
		Meta: entities.Metadata{
			Type: args[0],
			Name: args[1],
			MAC:  args[2]},
	}

	if err := json.NewEncoder(conn).Encode(request); err != nil {
		log.Errorf("askConfig(): Encode JSON: %s", err)
	}

	log.Println("Request:", request)

	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		log.Errorf("askConfig(): Decode JSON: %s", err)
	}

	if err != nil && response.IsEmpty() {
		panic("Connection has been closed by center")
	}

	dfc.updateConfig(response)

	go func() {
		for {
			defer func() {
				if r := recover(); r != nil {
					c.Close()
					log.Error("Initialization Failed")
				}
			} ()
			listenConfig(dfc, conn)
		}
	}()
}
