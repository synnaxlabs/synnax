package kv

import (
	"github.com/synnaxlabs/x/alamos"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"go.uber.org/zap"
	"time"
)

// Config is the configuration for the aspen kv service. For default values, see DefaultConfig().
type Config struct {
	// Cluster is the cluster that the DB will use to communicate with other databases.
	// [Required]
	Cluster cluster.Cluster
	// OperationsTransport is used to send key-value Operations between nodes.
	// [Required]
	OperationsTransport BatchTransport
	// FeedbackTransport is used to send gossip feedback between nodes.
	// [Required]
	FeedbackTransport FeedbackTransport
	// LeaseTransport is used to send leaseAlloc Operations between nodes.
	// [Required]
	LeaseTransport LeaseTransport
	// Logger is the witness of it all.
	// [Not Required]
	Logger *zap.SugaredLogger
	// Engine is the underlying key-value engine that DB writes its Operations to.
	// [Required]
	Engine kvx.DB
	// GossipInterval is how often a node initiates gossip with a peer.
	// [Not Required]
	GossipInterval time.Duration
	// Recovery threshold for the SIR gossip protocol i.e. how many times the node must send a redundant operation
	// for it to stop propagating it.
	//[Not Required]
	RecoveryThreshold int
}

func (cfg Config) Override(other Config) Config {
	cfg.Cluster = override.Nil(cfg.Cluster, other.Cluster)
	cfg.OperationsTransport = override.Nil(cfg.OperationsTransport, other.OperationsTransport)
	cfg.FeedbackTransport = override.Nil(cfg.FeedbackTransport, other.FeedbackTransport)
	cfg.LeaseTransport = override.Nil(cfg.LeaseTransport, other.LeaseTransport)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	cfg.Engine = override.Nil(cfg.Engine, other.Engine)
	cfg.GossipInterval = override.Numeric(cfg.GossipInterval, other.GossipInterval)
	cfg.RecoveryThreshold = override.Numeric(cfg.RecoveryThreshold, other.RecoveryThreshold)
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("kv")
	validate.NotNil(v, "Cluster", cfg.Cluster)
	validate.NotNil(v, "OperationsTransport", cfg.OperationsTransport)
	validate.NotNil(v, "FeedbackTransport", cfg.FeedbackTransport)
	validate.NotNil(v, "LeaseTransport", cfg.LeaseTransport)
	validate.NotNil(v, "Engine", cfg.Engine)
	return v.Error()
}

func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["recoveryThreshold"] = cfg.RecoveryThreshold
	report["gossipInterval"] = cfg.GossipInterval.String()
	report["operationsTransport"] = cfg.OperationsTransport.Report()
	report["feedbackTransport"] = cfg.FeedbackTransport.Report()
	report["leaseTransport"] = cfg.LeaseTransport.Report()
	return report
}

var DefaultConfig = Config{
	GossipInterval:    1 * time.Second,
	RecoveryThreshold: 5,
}
