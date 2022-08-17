package pledge

import (
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"time"
)

type Transport = freighter.Unary[node.ID, node.ID]

// Config is used for configuring a pledge based membership network.
type Config struct {
	// Transport is used for sending pledge information over the network.
	Transport Transport
	// RequestTimeout is the timeout for a peer to respond to a pledge or proposal request.
	// If the request is not responded to before the timeout, a new jury will be formed and the request will be retried.
	RequestTimeout time.Duration
	// RetryInterval sets the initial retry interval for a Pledge to a peer.
	RetryInterval time.Duration
	// PledgeInterval scale sets how quickly the time in-between retries will increase during a Pledge to a peer. For example,
	// a value of 2 would result in a retry interval of 1,2, 4, 8, 16, 32, 64, ... seconds.
	RetryScale float64
	// Logger is the witness of it all.
	Logger *zap.SugaredLogger
	// MaxProposals is the maximum number of proposals a responsible will make to a quorum before giving up.
	MaxProposals int
	// Experiment is where the gossip services saves its metrics and reports.
	Experiment alamos.Experiment
	// candidates is a Group of nodes to contact for as candidates for the formation of a jury.
	candidates func() node.Group
	// peerAddresses is a set of addresses a pledge can contact.
	peerAddresses []address.Address
}

func (cfg Config) Merge(def Config) Config {
	if cfg.Transport == nil {
		cfg.Transport = def.Transport
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = def.RequestTimeout
	}
	if cfg.RetryInterval == 0 {
		cfg.RetryInterval = def.RetryInterval
	}
	if cfg.RetryScale == 0 {
		cfg.RetryScale = def.RetryScale
	}
	if cfg.Logger == nil {
		cfg.Logger = def.Logger
	}
	if cfg.MaxProposals == 0 {
		cfg.MaxProposals = def.MaxProposals
	}
	if cfg.candidates == nil {
		cfg.candidates = def.candidates
	}
	if cfg.peerAddresses == nil {
		cfg.peerAddresses = def.peerAddresses
	}
	return cfg
}

func (cfg Config) Validate() error {
	if cfg.Transport == nil {
		return errors.AssertionFailedf("[pledge] - transport required")
	}
	return nil
}

// String returns a pretty printed string representation of the config.
func (cfg Config) String() string { return cfg.Report().String() }

// Report implements the alamos.Reporter interface.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	if cfg.Transport != nil {
		report["transport"] = cfg.Transport.Digest()
	} else {
		report["transport"] = "not provided"
	}
	report["requestTimeout"] = cfg.RequestTimeout
	report["pledgeBaseRetry"] = cfg.RetryInterval
	report["pledgeRetryScale"] = cfg.RetryScale
	report["maxProposals"] = cfg.MaxProposals
	report["peerAddresses"] = cfg.peerAddresses
	return report
}

func (cfg Config) LogArgs() []interface{} {
	return append([]interface{}{
		"requestTimeout", cfg.RequestTimeout,
		"pledgeBaseRetry", cfg.RetryInterval,
		"pledgeRetryScale", cfg.RetryScale,
		"maxProposals", cfg.MaxProposals,
		"peerAddresses", cfg.peerAddresses,
	}, cfg.Transport.Digest().LogArgs()...)
}

func DefaultConfig() Config {
	return Config{
		RequestTimeout: 5 * time.Second,
		RetryInterval:  1 * time.Second,
		RetryScale:     1.25,
		Logger:         zap.NewNop().Sugar(),
		MaxProposals:   10,
		peerAddresses:  []address.Address{},
	}
}
