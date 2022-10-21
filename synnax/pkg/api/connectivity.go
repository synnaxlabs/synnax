package api

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/errors"
)

// ConnectivityService is a simple service that allows a client to check their connection
// to the server.
type ConnectivityService struct{}

func NewConnectivityService(p Provider) *ConnectivityService {
	return &ConnectivityService{}
}

// Check does nothing except return a success response.
func (c *ConnectivityService) Check(ctx context.Context, _ types.Nil) (types.Nil, errors.Typed) {
	return types.Nil{}, errors.Nil
}
