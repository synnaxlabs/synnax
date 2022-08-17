package gossip

import (
	"github.com/arya-analytics/x/alamos"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"time"
)

// Config sets specific parameters for the gossip service. See DefaultConfig() for default values.
type Config struct {
	// Interval is the interval at which a node will gossip its state.
	// [Not Required]
	Interval time.Duration
	// Transport is the transport used to exchange gossip between nodes.
	// [Required]
	Transport Transport
	// Logger is the witness of it all.
	// [Not Required]
	Logger *zap.SugaredLogger
	// Experiment is where the gossip services saves its metrics and reports.
	// [Not Required]
	Experiment alamos.Experiment
}

func (cfg Config) Merge(def Config) Config {
	if cfg.Interval <= 0 {
		cfg.Interval = def.Interval
	}
	if cfg.Logger == nil {
		cfg.Logger = def.Logger
	}
	if cfg.Transport == nil {
		cfg.Transport = def.Transport
	}
	return cfg
}

func (cfg Config) Validate() error {
	if cfg.Transport == nil {
		return errors.AssertionFailedf("[gossip] - transport required")
	}
	return nil
}

func (cfg Config) LogArgs() []interface{} {
	return append([]interface{}{
		"interval",
		cfg.Interval,
	}, cfg.Transport.Digest().LogArgs()...)
}

// String returns a pretty printed string representation of the config.
func (cfg Config) String() string { return cfg.Report().String() }

// Report implements the alamos.Reporter interface.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["interval"] = cfg.Interval
	if cfg.Transport != nil {
		report["transport"] = cfg.Transport.Digest()
	} else {
		report["transport"] = "not provided"
	}
	return report
}

func DefaultConfig() Config {
	return Config{
		Interval: 1 * time.Second,
		Logger:   zap.NewNop().Sugar(),
	}
}
