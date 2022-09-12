package gossip

import (
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/override"
	"github.com/arya-analytics/x/validate"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
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
func (cfg Config) Override(other Config) Config {
	cfg.Interval = override.Numeric(cfg.Interval, other.Interval)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.Store = override.Nil(cfg.Store, other.Store)
	return cfg
}

// Validate implements the config.Config interface.
func (cfg Config) Validate() error {
	v := validate.New("gossip")
	validate.NotNil(v, "transport", cfg.Transport)
	validate.NotNil(v, "store", cfg.Store)
	validate.Positive(v, "interval", cfg.Interval)
	validate.NotNil(v, "logger", cfg.Logger)
	return v.Error()
}

// Report implements the alamos.Reporter interface. Assumes the config is valid.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["interval"] = cfg.Interval
	report["transport"] = cfg.Transport.Report()
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
