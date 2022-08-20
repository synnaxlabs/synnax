package gossip

import (
	"github.com/arya-analytics/aspen/internal/cluster/store"
	"github.com/arya-analytics/x/alamos"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"time"
)

// Config sets specific parameters for the gossip service. See DefaultConfig
// for default values. It implements the config.Config interface.
type Config struct {
	// Transport is the transport used to exchange gossip between nodes.
	// [Required]
	Transport Transport
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
func (cfg Config) Override(override Config) Config {
	if override.Interval > 0 {
		cfg.Interval = override.Interval
	}
	if override.Logger != nil {
		cfg.Logger = override.Logger
	}
	if override.Transport != nil {
		cfg.Transport = override.Transport
	}
	if override.Store != nil {
		cfg.Store = override.Store
	}
	return cfg
}

// Validate implements the config.Config interface.
func (cfg Config) Validate() error {
	if cfg.Transport == nil {
		return errors.New("[gossip] - transport required")
	}
	if cfg.Store == nil {
		return errors.New("[gossip] - store required")
	}
	if cfg.Interval <= 0 {
		return errors.New("[gossip] - interval must be positive")
	}
	if cfg.Logger == nil {
		return errors.New("[gossip] - logger required")
	}
	return nil
}

// Report implements the alamos.Reporter interface. Assumes the config is valid.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["interval"] = cfg.Interval
	report["transport"] = cfg.Transport.Digest()
	return report
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
