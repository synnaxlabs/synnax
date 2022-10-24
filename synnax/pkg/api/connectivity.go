package api

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/errors"
)

// ConnectivityService is a simple service that allows a client to check their connection
// to the server.
type ConnectivityService struct {
	clusterProvider
}

func NewConnectivityService(p Provider) *ConnectivityService {
	return &ConnectivityService{clusterProvider: p.cluster}
}

// ConnectivityCheckResponse is returned by the ConnectivityService.Check method.
type ConnectivityCheckResponse struct {
	ClusterKey string `json:"cluster_key"`
}

// Check does nothing except return a success response.
func (c *ConnectivityService) Check(ctx context.Context, _ types.Nil) (ConnectivityCheckResponse, errors.Typed) {
	return ConnectivityCheckResponse{
		ClusterKey: c.clusterProvider.cluster.Key().String(),
	}, errors.Nil
}
