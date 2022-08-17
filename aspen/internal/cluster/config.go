package cluster

import (
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	pledge_ "github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/kv"
	"go.uber.org/zap"
	"time"
)

const FlushOnEvery = time.Duration(-1)

type Config struct {
	// Storage is a key-value storage backend for the cluster. Cluster will flush changes to its state to this backend
	// based on Config.StorageFlushInterval. Join will also attempt to load an existing cluster from this backend.
	// If Config.Storage is not provided, Cluster state will only be stored in memory.
	Storage kv.DB
	// StorageKey is the key used to store the cluster state in the backend.
	StorageKey []byte
	// StorageFlushInterval	is the interval at which the cluster state is flushed to the backend. If this is set to FlushOnEvery,
	// the cluster state will be flushed every time a change is made.
	StorageFlushInterval time.Duration
	// Logger is the witness of it all.
	Logger *zap.SugaredLogger
	// Gossip is the configuration for propagating Cluster state through gossip. See the gossip package for more details
	// on how to configure this.
	Gossip gossip.Config
	// Pledge is the configuration for pledging to the cluster upon a Join call. See the pledge package for more details
	// on how to configure this.
	Pledge pledge_.Config
	// Experiment is where the pledge services saves its metrics and reports.
	Experiment alamos.Experiment
}

func (cfg Config) Merge(def Config) Config {
	if cfg.Logger == nil {
		cfg.Logger = def.Logger
	}
	if cfg.StorageFlushInterval == 0 {
		cfg.StorageFlushInterval = def.StorageFlushInterval
	}
	if len(cfg.StorageKey) == 0 {
		cfg.StorageKey = def.StorageKey
	}

	// |||| PLEDGE ||||

	if cfg.Pledge.Logger == nil {
		cfg.Pledge.Logger = cfg.Logger.Named("pledge")
	}
	if cfg.Pledge.Experiment == nil {
		cfg.Pledge.Experiment = cfg.Experiment
	}

	// |||| GOSSIP ||||

	if cfg.Gossip.Logger == nil {
		cfg.Gossip.Logger = cfg.Logger.Named("gossip")
	}
	if cfg.Gossip.Experiment == nil {
		cfg.Gossip.Experiment = cfg.Experiment
	}

	return cfg
}

func (cfg Config) Validate() error {
	c := errutil.NewCatchSimple()
	c.Exec(cfg.Pledge.Validate)
	c.Exec(cfg.Gossip.Validate)
	return c.Error()
}

// String returns a pretty printed representation of the config.
func (cfg Config) String() string { return cfg.Report().String() }

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

func DefaultConfig() Config {
	return Config{
		Pledge:               pledge_.DefaultConfig(),
		StorageKey:           []byte("aspen.cluster"),
		Logger:               zap.NewNop().Sugar(),
		Gossip:               gossip.DefaultConfig(),
		StorageFlushInterval: 1 * time.Second,
	}
}
