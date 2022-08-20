package pledge

import (
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/config"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"time"
)

type Transport = freighter.Unary[node.ID, node.ID]

// Config is used for configuring a pledge based membership network. It implements
// the config.Config interface.
type Config struct {
	// Candidates is a Group of nodes to contact as candidates for the formation
	// of a jury.
	// [Required]
	Candidates func() node.Group
	// Peers is a set of addresses a pledge can contact.
	// [Required]
	Peers []address.Address
	// Transport is used for sending pledge information over the network.
	// [Required]
	Transport Transport
	// RequestTimeout is the timeout for a peer to respond to a pledge or proposal
	// request. If the request is not responded to before the timeout, a new jury
	// will be formed and the request will be retried.
	RequestTimeout time.Duration
	// RetryInterval sets the initial retry interval for a Pledge to a peer.
	RetryInterval time.Duration
	// MaxProposals is the maximum number of proposals a responsible node will make
	// to a quorum before giving up.
	MaxProposals int
	// PledgeInterval scale sets how quickly the time in-between retries will
	// increase during a Pledge to a peer. For example, a value of 2 would result
	// in a retry interval of 1,2, 4, 8, 16, 32, 64, ... seconds.
	RetryScale float64
	// Logger is the witness of it all.
	Logger *zap.SugaredLogger
	// Experiment is where the gossip services saves its metrics and reports.
	Experiment alamos.Experiment
}

var _ config.Config[Config] = Config{}

// Override implements the config.Config interface.
func (cfg Config) Override(override Config) Config {
	if override.Transport != nil {
		cfg.Transport = override.Transport
	}
	if override.RequestTimeout != 0 {
		cfg.RequestTimeout = override.RequestTimeout
	}
	if override.RetryInterval != 0 {
		cfg.RetryInterval = override.RetryInterval
	}
	if override.RetryScale != 0 {
		cfg.RetryScale = override.RetryScale
	}
	if override.Logger != nil {
		cfg.Logger = override.Logger
	}
	if override.MaxProposals != 0 {
		cfg.MaxProposals = override.MaxProposals
	}
	if override.Candidates != nil {
		cfg.Candidates = override.Candidates
	}
	if len(override.Peers) > 0 {
		cfg.Peers = override.Peers
	}
	return cfg
}

// Validate implements the config.Config interface.
func (cfg Config) Validate() error {
	if cfg.Transport == nil {
		return errors.New("[pledge] - transport required")
	}
	if cfg.RequestTimeout == 0 {
		return errors.New("[pledge] - request timeout must be non-zero")
	}
	if cfg.RetryScale < 1 {
		return errors.New("[pledge] - retry scale must be >= 1")
	}
	if cfg.MaxProposals == 0 {
		return errors.New("[pledge] - max proposals must be non-zero")
	}
	if cfg.Candidates == nil {
		return errors.New("[pledge] - candidates required")
	}
	if cfg.Logger == nil {
		return errors.New("[pledge] - logger required")
	}
	return nil
}

// Report implements the alamos.Reporter interface. Assumes the Config is valid.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["Transport"] = cfg.Transport.Report()
	report["requestTimeout"] = cfg.RequestTimeout
	report["pledgeBaseRetry"] = cfg.RetryInterval
	report["pledgeRetryScale"] = cfg.RetryScale
	report["maxProposals"] = cfg.MaxProposals
	report["Peers"] = cfg.Peers
	return report
}

var (
	DefaultConfig = Config{
		RequestTimeout: 5 * time.Second,
		RetryInterval:  1 * time.Second,
		RetryScale:     1.25,
		Logger:         zap.NewNop().Sugar(),
		MaxProposals:   10,
		Peers:          []address.Address{},
	}
	FastConfig = DefaultConfig.Override(Config{
		RequestTimeout: 50 * time.Millisecond,
		RetryInterval:  10 * time.Millisecond,
		RetryScale:     1.125,
	})
	BlazingFastConfig = DefaultConfig.Override(Config{
		RequestTimeout: 5 * time.Millisecond,
		RetryInterval:  1 * time.Microsecond,
	})
)
