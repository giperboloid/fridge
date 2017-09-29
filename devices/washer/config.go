package config

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/KharkivGophers/device-smart-house/error"
	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/devicems/entities"
)

type WasherConfig struct {
	sync.Mutex
	Temperature    int64
	WashTime       int64
	WashTurnovers  int64
	RinseTime      int64
	RinseTurnovers int64
	SpinTime       int64
	SpinTurnovers  int64
	subsPool       map[string]chan struct{}
}

func NewWasherConfig() *WasherConfig {
	conf := &WasherConfig{}
	conf.subsPool = make(map[string]chan struct{})

	return conf
}

func (wc *WasherConfig) GetTemperature() int64 {
	wc.Mutex.Lock()
	defer wc.Mutex.Unlock()
	return wc.Temperature
}
func (wc *WasherConfig) SetTemperature(b int64) {
	wc.Mutex.Lock()
	wc.Temperature = b
	wc.Mutex.Unlock()
}

func (wc *WasherConfig) GetWashTime() int64 {
	wc.Mutex.Lock()
	defer wc.Mutex.Unlock()
	return wc.WashTime
}
func (wc *WasherConfig) SetWashTime(b int64) {
	wc.Mutex.Lock()
	wc.WashTime = b
	wc.Mutex.Unlock()
}

func (wc *WasherConfig) GetWashTurnovers() int64 {
	wc.Mutex.Lock()
	defer wc.Mutex.Unlock()
	return wc.WashTurnovers
}
func (wc *WasherConfig) SetWashTurnovers(b int64) {
	wc.Mutex.Lock()
	wc.WashTurnovers = b
	wc.Mutex.Unlock()
}

func (wc *WasherConfig) GetRinseTime() int64 {
	wc.Mutex.Lock()
	defer wc.Mutex.Unlock()
	return wc.RinseTime
}
func (wc *WasherConfig) SetRinseTime(b int64) {
	wc.Mutex.Lock()
	wc.RinseTime = b
	wc.Mutex.Unlock()
}

func (wc *WasherConfig) GetRinseTurnovers() int64 {
	wc.Mutex.Lock()
	defer wc.Mutex.Unlock()
	return wc.RinseTurnovers
}
func (wc *WasherConfig) SetRinseTurnovers(b int64) {
	wc.Mutex.Lock()
	wc.RinseTurnovers = b
	wc.Mutex.Unlock()
}

func (wc *WasherConfig) GetSpinTime() int64 {
	wc.Mutex.Lock()
	defer wc.Mutex.Unlock()
	return wc.SpinTime
}
func (wc *WasherConfig) SetSpinTime(b int64) {
	wc.Mutex.Lock()
	wc.SpinTime = b
	wc.Mutex.Unlock()
}

func (wc *WasherConfig) GetSpinTurnovers() int64 {
	wc.Mutex.Lock()
	defer wc.Mutex.Unlock()
	return wc.SpinTurnovers
}
func (wc *WasherConfig) SetSpinTurnovers(b int64) {
	wc.Mutex.Lock()
	wc.SpinTurnovers = b
	wc.Mutex.Unlock()
}

func (wc *WasherConfig) updateWasherConfig(c entities.WasherConfig) {
	wc.Temperature = c.Temperature

	wc.WashTime = c.WashTime

	wc.WashTurnovers = c.WashTurnovers

	wc.RinseTime = c.RinseTime

	wc.RinseTurnovers = c.RinseTurnovers

	wc.SpinTime = c.SpinTime

	wc.SpinTurnovers = c.SpinTurnovers

	log.Warn("New Configuration:")
	log.Warn("Temperature: ", wc.Temperature, "; WashTime: ", wc.WashTime, "; WashTurnovers: ", wc.WashTurnovers,
		"; RinseTime: ", wc.RinseTime, "; RinseTurnovers: ", wc.RinseTurnovers, "; SpinTime: ", wc.SpinTime,
		"; SpinTurnovers: ", wc.SpinTurnovers)
}

func (wc *WasherConfig) RequestWasherConfig(connType string, host string, port string, args []string) entities.WasherConfig {
	conn, err := net.Dial(connType, host+":"+port)
	for err != nil {
		log.Error("Can't connect to the server: " + host + ":" + port)
		panic("No center found!")
	}

	var response entities.WasherConfig
	var request entities.WasherRequest

	request = entities.WasherRequest{
		Action: "config",
		Meta: entities.Metadata{
			Type: args[0],
			Name: args[1],
			MAC:  args[2]},
	}

	err = json.NewEncoder(conn).Encode(request)
	error.CheckError("askConfig(): Encode JSON", err)

	log.Println("Request:", request)

	err = json.NewDecoder(conn).Decode(&response)
	error.CheckError("askConfig(): Decode JSON", err)

	if err != nil {
		panic("Connection has been close by central server!")
	}

	return response
}

func (wc *WasherConfig) SendWasherRequests(connType string, host string, port string, c *entities.Control, args []string, nextStep chan struct{}) {

	ticker := time.NewTicker(time.Second)
	defer func() {
		if r := recover(); r != nil {
			log.Error(r)
		}
	}()
	response := wc.RequestWasherConfig(connType, host, port, args)
	log.Println("Response:", response)

	for {
		select {
		case <-ticker.C:
			switch response.IsEmpty() {
			case true:
				log.Println("Response:", response)
				defer func() {
					if r := recover(); r != nil {
						log.Error(r)
						c.Close()
					}
				}()
				wc.RequestWasherConfig(connType, host, port, args)
			default:
				wc.updateWasherConfig(response)
				ticker.Stop()
				nextStep <- struct{}{}
				return
			}
		}
	}
}

//func (wc *WasherConfig) AddSubIntoPool(key string, value chan struct{}) {
//	wc.Mutex.Lock()
//	wc.subsPool[key] = value
//	wc.Mutex.Unlock()
//}
//
//func (wc *WasherConfig) RemoveSubFromPool(key string) {
//	wc.Mutex.Lock()
//	delete(wc.subsPool, key)
//	wc.Mutex.Unlock()
//}
//
//func publishWasherConfig(wc *WasherConfig) {
//	for _, v := range wc.subsPool {
//		v <- struct{}{}
//	}
//}
