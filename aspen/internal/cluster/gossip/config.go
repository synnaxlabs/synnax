package gossip

import (
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"time"
)

// Config sets specific parameters for the gossip service. See DefaultConfig
// for default values. It implements the config.Config interface.
type Config struct {
	// TransportClient is the transport used to exchange gossip between nodes.
	// [Required]
	TransportClient TransportClient
	// TransportServer is the transport used to exchange gossip between nodes.
	// [Required]
	TransportServer TransportServer
	// Store is where cluster state will be synchronized to and from.
	// [Required]
	Store store.Store
	// Interval is the interval at which a node will gossip its state.
	Interval time.Duration
	// Logger is the witness of it all.
	Logger *zap.SugaredLogger
	// Experiment is where the gossip services saves its metrics and reports.
	Experiment alamos.Experiment
}

// Override implements the config.Config interface.
func (cfg Config) Override(other Config) Config {
	cfg.Interval = override.Numeric(cfg.Interval, other.Interval)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	cfg.TransportClient = override.Nil(cfg.TransportClient, other.TransportClient)
	cfg.Store = override.Nil(cfg.Store, other.Store)
	return cfg
}

// Validate implements the config.Config interface.
func (cfg Config) Validate() error {
	v := validate.New("gossip")
	validate.NotNil(v, "TransportClient", cfg.TransportClient)
	validate.NotNil(v, "TransportServer", cfg.TransportServer)
	validate.NotNil(v, "Store", cfg.Store)
	validate.Positive(v, "Interval", cfg.Interval)
	validate.NotNil(v, "Logger", cfg.Logger)
	return v.Error()
}

// Report implements the alamos.Reporter interface. Assumes the config is valid.
func (cfg Config) Report() alamos.Report {
	return alamos.Report{
		"interval":        cfg.Interval,
		"transportClient": cfg.TransportClient.Report(),
		"transportServer": cfg.TransportServer.Report(),
	}
}

var (
	DefaultConfig = Config{
		Interval: 1 * time.Second,
		Logger:   zap.NewNop().Sugar(),
	}
	FastConfig = DefaultConfig.Override(Config{
		Interval: 50 * time.Millisecond,
	})
)
