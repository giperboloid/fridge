// Package entities provides definitions for common basic entities
// and some of their functions.
package entities

import "time"

// Server is used to store IP and open port of a remote server.
type Server struct {
	Host string
	Port string
}

// DevMeta is used to store device metadata: it's type, name (model) and MAC.
type DevMeta struct {
	Type string
	Name string
	MAC  string
}

// ServicesController is used to store StopChan that allows to terminate
// all the services that listen the channel.
type ServicesController struct {
	StopChan chan struct{}
}

// Terminate closes StopChan to signal all the services to shutdown.
func (c *ServicesController) Terminate() {
	select {
	case <-c.StopChan:
	default:
		close(c.StopChan)
	}
}

// Wait waits until StopChan will be closed and then makes a pause for 3 seconds
// in order to give time for all the services shutdown gracefully.
func (c *ServicesController) Wait() {
	<-c.StopChan
	<-time.NewTimer(time.Second * 3).C
}
