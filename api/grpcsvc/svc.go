package grpcsvc

import (
	"net"
	"time"

	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"github.com/giperboloid/fridgems/services"
	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type GRPCConfig struct {
	ConfigService *services.ConfigService
	Reconnect     *time.Ticker
	Server        entities.Server
}

func Init(c GRPCConfig) {
	s := newFridgeService(c)
	go s.listen()
}

func newFridgeService(c GRPCConfig) *API{
	return &API {
		Config: c.ConfigService,
		Reconnect: c.Reconnect,
		Server: c.Server,
	}
}

type API struct {
	Config    *services.ConfigService
	Reconnect *time.Ticker
	Server    entities.Server
}

func (a *API) listen() {
	ln, err := net.Listen("tcp", a.Server.Host+":"+a.Server.Port)
	if err != nil {
		logrus.Errorf("API: listen(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range a.Reconnect.C {
			ln, err = net.Listen("tcp", a.Server.Host+":"+a.Server.Port)
			if err != nil {
				logrus.Errorf("API: listen(): Listen() has failed: %s", err)
			}
		}
		a.Reconnect.Stop()
	}

	gs := grpc.NewServer()
	pb.RegisterDevServiceServer(gs, a)
	gs.Serve(ln)
}

func (a *API) PatchDevConfig(ctx context.Context, in *pb.PatchDevConfigRequest) (*pb.PatchDevConfigResponse, error) {
	return a.Config.PatchDevConfig(ctx, in)
}
