package pledge

import (
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"time"
)

type (
	TransportClient = freighter.UnaryClient[node.ID, node.ID]
	TransportServer = freighter.UnaryServer[node.ID, node.ID]
)

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
	// TransportClient is used for sending pledge information over the network.
	// [Required]
	TransportClient TransportClient
	// TransportServer is used for receiving pledge information over the network.
	// [Required]
	TransportServer TransportServer
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
func (cfg Config) Override(other Config) Config {
	cfg.TransportClient = override.Nil(cfg.TransportClient, other.TransportClient)
	cfg.RequestTimeout = override.Numeric(cfg.RequestTimeout, other.RequestTimeout)
	cfg.RetryInterval = override.Numeric(cfg.RetryInterval, other.RetryInterval)
	cfg.RetryScale = override.Numeric(cfg.RetryScale, other.RetryScale)
	cfg.MaxProposals = override.Numeric(cfg.MaxProposals, other.MaxProposals)
	cfg.Logger = override.Nil[*zap.SugaredLogger](cfg.Logger, other.Logger)
	cfg.Candidates = override.Nil(cfg.Candidates, other.Candidates)
	cfg.Peers = override.Slice(cfg.Peers, other.Peers)
	return cfg
}

// Validate implements the config.Config interface.
func (cfg Config) Validate() error {
	v := validate.New("pledge")
	validate.NotNil(v, "TransportClient", cfg.TransportClient)
	validate.Positive(v, "RequestTimeout", cfg.RequestTimeout)
	validate.GreaterThanEq(v, "RetryScale", cfg.RetryScale, 1)
	validate.NonZero(v, "MaxProposals", cfg.MaxProposals)
	validate.NotNil(v, "Candidates", cfg.Candidates)
	return v.Error()
}

// Report implements the alamos.Reporter interface. Assumes the Config is valid.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["transportClient"] = cfg.TransportClient.Report()
	report["transportServer"] = cfg.TransportServer.Report()
	report["requestTimeout"] = cfg.RequestTimeout
	report["pledgeBaseRetry"] = cfg.RetryInterval
	report["pledgeRetryScale"] = cfg.RetryScale
	report["maxProposals"] = cfg.MaxProposals
	report["peers"] = cfg.Peers
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
