package entities

import "time"

type Server struct {
	Host string
	Port string
}

type DevMeta struct {
	Type string
	Name string
	MAC  string
}

type FridgeConfig struct {
	TurnedOn    bool
	CollectFreq int64
	SendFreq    int64
}

type ServicesController struct {
	StopChan chan struct{}
}

func (c *ServicesController) Wait() {
	<-c.StopChan
	<-time.NewTimer(time.Second * 3).C
}

func (c *ServicesController) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}
