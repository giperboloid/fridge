package api

import (
	"github.com/giperboloid/fridgems/services"
	"golang.org/x/net/context"
	"github.com/giperboloid/fridgems/pb"
)

type API struct {
	config services.ConfigService
}

func (a *API) PatchDevConfig(ctx context.Context, in *pb.PatchDevConfigRequest) (*pb.PatchDevConfigResponse, error) {
	return nil, nil
}