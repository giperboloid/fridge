package api

import (
	"net"
	"time"

	"github.com/giperboloid/fridgems/entities"
	"github.com/giperboloid/fridgems/pb"
	"github.com/giperboloid/fridgems/services"
	"github.com/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type API struct {
	Config    *services.ConfigService
	Reconnect *time.Ticker
	Server    entities.Server
}

func NewAPI(c *services.ConfigService, r *time.Ticker, s entities.Server) *API{
	return &API {
		Config: c,
		Reconnect: r,
		Server: s,
	}
}

func (a *API) Listen() {
	ln, err := net.Listen("tcp", a.Server.Host+":"+a.Server.Port)
	if err != nil {
		logrus.Errorf("listenConfigPatch(): Listen() has failed: %s", err)
	}

	for err != nil {
		for range a.Reconnect.C {
			ln, err = net.Listen("tcp", a.Server.Host+":"+a.Server.Port)
			if err != nil {
				logrus.Errorf("listenConfigPatch(): Listen() has failed: %s", err)
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
