package fridge

import (
	"encoding/json"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"time"
	"golang.org/x/net/context"
	"encoding/binary"
	"bytes"
	"fmt"
	"google.golang.org/grpc"
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

func (c *Configuration) SetInitConfig(s entities.Server, m *entities.DevMeta, ctrl *entities.RoutinesController) {
	pbic := &pb.SetDevInitConfigRequest{
		Time:   time.Now().UnixNano(),
		Meta: &pb.DevMeta{
			Type: m.Type,
			Name: m.Name,
			Mac:  m.MAC,
		},
	}

	conn := dial(s)
	defer conn.Close()

	client := pb.NewCenterServiceClient(conn)

	r, err := client.SetDevInitConfig(context.Background(), pbic)
	if err != nil {
		log.Error("SetInitConfig(): SetDevInitConfig() has failed: ", err)
		panic("init config hasn't been received")
	}

	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, r.Config); err != nil {
		panic(err)
	}

	var fc entities.FridgeConfig
	if err := json.NewDecoder(buf).Decode(&fc); err != nil {
		panic("init config decoding has failed")
	}

	log.Infof("init config: %+v", fc)
	c.update(&fc)

	go c.listenConfigPatch(s, time.NewTicker(time.Second * 3))
}

func (c *Configuration) listenConfigPatch(s entities.Server, r *time.Ticker) {
	ln, err := net.Listen("tcp", "127.0.0.1"+":"+fmt.Sprint("4000"))
	if err != nil {
		log.Errorf("listenConfigPatch(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range r.C {
			ln, err = net.Listen("tcp", "127.0.0.1"+":"+fmt.Sprint("4000"))
			if err != nil {
				log.Errorf("listenConfigPatch(): Listen() has failed: %s", err)
			}
		}
		r.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterDevServiceServer(gs, c)
	gs.Serve(ln)
}

func (c *Configuration) PatchDevConfig(ctx context.Context, r *pb.PatchDevConfigRequest) (*pb.PatchDevConfigResponse, error) {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, r.Config); err != nil {
		panic(err)
	}

	var fc entities.FridgeConfig
	if err := json.NewDecoder(buf).Decode(&fc); err != nil {
		panic("config patch decoding has failed")
	}

	log.Infof("config patch: %+v", fc)
	c.update(&fc)
	return &pb.PatchDevConfigResponse{"OK"}, nil
}

func (c *Configuration) update(nfc *entities.FridgeConfig) {
	if nfc.TurnedOn && !c.TurnedOn {
		log.Info("fridge is turned on")
	} else if !nfc.TurnedOn && c.TurnedOn {
		log.Info("fridge is turned off")
	}

	c.TurnedOn = nfc.TurnedOn
	c.CollectFreq = nfc.CollectFreq
	c.SendFreq = nfc.SendFreq
}

func (c *Configuration) publishConfigPatch() {
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
