package cluster

import (
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	pledge_ "github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/kv"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"time"
)

const FlushOnEvery = -1 * time.Second

type Config struct {
	// HostAddress is the reachable address of the host node.
	// [REQUIRED]
	HostAddress address.Address
	// Storage is a key-value storage backend for the cluster. Cluster will flush
	// changes to its state to this backend based on Config.StorageFlushInterval.
	// Join will also attempt to load an existing cluster from this backend.
	// If Config.Storage is not provided, Cluster state will only be stored in memory.
	Storage kv.DB
	// StorageKey is the key used to store the cluster state in the backend.
	StorageKey []byte
	// StorageFlushInterval	is the interval at which the cluster state is flushed
	// to the backend. If this is set to FlushOnEvery, the cluster state is flushed on
	// every change.
	StorageFlushInterval time.Duration
	// Logger is the witness of it all.
	Logger *zap.SugaredLogger
	// Gossip is the configuration for propagating Cluster state through gossip.
	// See the gossip package for more details on how to configure this.
	Gossip gossip.Config
	// Pledge is the configuration for pledging to the cluster upon a Join call.
	// See the pledge package for more details on how to configure this.
	Pledge pledge_.Config
	// Experiment is where the pledge services saves its metrics and reports.
	Experiment alamos.Experiment
	// EncoderDecoder is the encoder/decoder to use for encoding and decoding the
	// cluster state.
	EncoderDecoder binary.EncoderDecoder
}

func (cfg Config) Override(override Config) Config {
	if override.HostAddress != "" {
		cfg.HostAddress = override.HostAddress
	}

	if override.Logger != nil {
		cfg.Logger = override.Logger
		cfg.Pledge.Logger = override.Logger
		cfg.Gossip.Logger = override.Logger
	}

	if override.EncoderDecoder != nil {
		cfg.EncoderDecoder = override.EncoderDecoder
	}

	if override.StorageFlushInterval != 0 {
		cfg.StorageFlushInterval = override.StorageFlushInterval
	}
	if len(override.StorageKey) != 0 {
		cfg.StorageKey = override.StorageKey
	}
	if override.Storage != nil {
		cfg.Storage = override.Storage
	}

	if override.Experiment != nil {
		cfg.Experiment = override.Experiment
		cfg.Pledge.Experiment = override.Experiment
		cfg.Gossip.Experiment = override.Experiment
	}

	cfg.Gossip = cfg.Gossip.Override(override.Gossip)
	cfg.Pledge = cfg.Pledge.Override(override.Pledge)

	return cfg
}

func (cfg Config) Validate() error {
	if cfg.HostAddress == "" {
		return errors.New("[cluster] - HostAddress is required")
	}
	c := errutil.NewCatch()
	c.Exec(cfg.Gossip.Validate)
	c.Exec(cfg.Pledge.Validate)
	return c.Error()
}

// Report implements the alamos.Reporter interface.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	if cfg.Storage != nil {
		report["storage"] = cfg.Storage.String()
	} else {
		report["storage"] = "not provided"
	}
	report["storageKey"] = string(cfg.StorageKey)
	report["storageFlushInterval"] = cfg.StorageFlushInterval
	return report
}

var (
	DefaultConfig = Config{
		Pledge:               pledge_.DefaultConfig,
		StorageKey:           []byte("aspen.cluster"),
		Logger:               zap.NewNop().Sugar(),
		Gossip:               gossip.DefaultConfig,
		StorageFlushInterval: 1 * time.Second,
		EncoderDecoder:       &binary.GobEncoderDecoder{},
	}
	FastConfig = DefaultConfig.Override(Config{
		Pledge: pledge_.FastConfig,
		Gossip: gossip.FastConfig,
	})
	BlazingFastConfig = DefaultConfig.Override(Config{
		Pledge: pledge_.BlazingFastConfig,
		Gossip: gossip.FastConfig,
	})
)
