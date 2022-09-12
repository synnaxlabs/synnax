package core

import (
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/arya-analytics/freighter/fgrpc"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/config"
	"github.com/arya-analytics/x/override"
	"github.com/arya-analytics/x/validate"
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

var _ config.Config[Config] = Config{}

func (cfg Config) Override(other Config) Config {
	cfg.AdvertiseAddress = override.String(cfg.AdvertiseAddress, other.AdvertiseAddress)
	cfg.PeerAddresses = override.Slice(cfg.PeerAddresses, other.PeerAddresses)
	cfg.Experiment = override.Nil(cfg.Experiment, other.Experiment)
	cfg.Pool = override.Nil(cfg.Pool, other.Pool)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	cfg.Storage = cfg.Storage.Override(other.Storage)
	cfg.Transports = override.Nil(cfg.Transports, other.Transports)
	cfg.Storage.Logger = cfg.Logger.Named("storage")
	cfg.Storage.Experiment = cfg.Experiment
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("distribution.core")
	validate.NotNil(v, "Pool", cfg.Pool)
	return v.Error()
}

var DefaultConfig = Config{
	Logger:  zap.NewNop(),
	Storage: storage.DefaultConfig,
}
