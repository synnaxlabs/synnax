package core

import (
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/freighter/fgrpc"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

type Config struct {
	// AdvertiseAddress is the address the distribution layer will advertise to the rest of the nodes in the cluster.
	AdvertiseAddress address.Address
	// PeerAddresses is a list of addresses of other nodes to contact in the cluster for bootstrapping.
	// If no addresses are provided and storage is empty, the distribution layer will bootstrap a new cluster.
	// If a cluster already exists in storage, the addresses in this list will be ignored.
	PeerAddresses []address.Address
	Experiment    alamos.Experiment
	// Logger is the witness of it all.
	Logger *zap.Logger
	// Pool is a pool for grpc connections to other nodes in the cluster.
	Pool *fgrpc.Pool
	// Storage is the storage configuration to use for the node.
	Storage    storage.Config
	Transports *[]fgrpc.BindableTransport
}

func (cfg Config) Merge(def Config) Config {

	if cfg.Logger == nil {
		cfg.Logger = def.Logger
	}

	// |||| STORAGE ||||

	if cfg.Storage.Logger == nil {
		cfg.Storage.Logger = cfg.Logger.Named("storage")
	}
	if cfg.Storage.Experiment == nil {
		cfg.Storage.Experiment = cfg.Experiment
	}

	return cfg
}

func (cfg Config) Validate() error {
	if cfg.Pool == nil {
		return errors.AssertionFailedf("[distribution] - pool required")
	}
	return nil
}

func DefaultConfig() Config {
	return Config{
		Logger:  zap.NewNop(),
		Storage: storage.DefaultConfig(),
	}
}
