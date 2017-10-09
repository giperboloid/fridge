package entities

import "time"

type Server struct {
	Host string
	Port string
}

type TransferConn struct {
	Server   Server
	ConnType string
}

type DevMeta struct {
	Type string `json:"type"`
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

type FridgeData struct {
	TempCam1 map[int64]float32 `json:"tempCam1"`
	TempCam2 map[int64]float32 `json:"tempCam2"`
}

type FridgeRequest struct {
	Action string     `json:"action"`
	Time   int64      `json:"time"`
	Meta   DevMeta   `json:"meta"`
	Data   FridgeData `json:"data"`
}

type Response struct {
	Descr string `json:"descr"`
}

type FridgeConfig struct {
	TurnedOn    bool   `json:"turnedOn"`
	CollectFreq int64  `json:"collectFreq"`
	SendFreq    int64  `json:"sendFreq"`
	MAC         string `json:"mac"`
}

func (fc *FridgeConfig) IsEmpty() bool {
	if fc.CollectFreq == 0 && fc.SendFreq == 0 && fc.MAC == "" && fc.TurnedOn == false {
		return true
	}
	return false
}

type FridgeGenerData struct {
	Time int64
	Data float32
}

type CollectFridgeData struct {
	CBot chan FridgeGenerData
	CTop chan FridgeGenerData
	ReqChan chan FridgeRequest
}

type RoutinesController struct {
	StopChan chan struct{}
}

func (c *RoutinesController) Wait() {
	<-c.StopChan
	<-time.NewTimer(time.Second * 3).C
}

func (c *RoutinesController) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
