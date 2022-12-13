package core

import (
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Config struct {
	// AdvertiseAddress is the address the distribution layer will advertise to the rest of the nodes in the Cluster.
	AdvertiseAddress address.Address
	// PeerAddresses is a list of addresses of other nodes to contact in the Cluster for bootstrapping.
	// If no addresses are provided and storage is empty, the distribution layer will bootstrap a new Cluster.
	// If a Cluster already exists in storage, the addresses in this list will be ignored.
	PeerAddresses []address.Address
	Experiment    alamos.Experiment
	// Logger is the witness of it all.
	Logger *zap.Logger
	// Pool is a pool for grpc connections to other nodes in the Cluster.
	Pool *fgrpc.Pool
	// Storage is the storage configuration to use for the node.
	Storage storage.Config
	// Transports is a list of transports the distribution uses for communication.
	// These Transports must be bound to the node's grpc server.
	Transports *[]fgrpc.BindableTransport
}

var _ config.Config[Config] = Config{}

// Override implements Config.
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

// Validate implements Config.
func (cfg Config) Validate() error {
	v := validate.New("distribution.core")
	validate.NotNil(v, "pool", cfg.Pool)
	return v.Error()
}

var DefaultConfig = Config{
	Logger:  zap.NewNop(),
	Storage: storage.DefaultConfig,
}
