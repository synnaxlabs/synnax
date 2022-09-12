package aspen

import (
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	grpct "github.com/synnaxlabs/aspen/transport/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

type Option func(*options)

type options struct {
	// dirname is the directory where aspen will store its data.
	// this option is ignored if a custom kv.Config.Engine is set.
	dirname string
	// addr sets the address for the host node.
	addr address.Address
	// peerAddresses sets the addresses for the peers of the host node.
	peerAddresses []address.Address
	// cluster gives the configuration for gossiping cluster state.
	cluster cluster.Config
	// kv gives the configuration for KV options.
	kv kv.Config
	// externalKV is a boolean flag indicating whether the caller provided an external
	// key-value engine. If so, aspen will not close the engine when it shuts down.
	externalKV bool
	// fs sets the filesystem to be used for storing data. This option is ignored
	// if a custom kv.Config.Engine is set.
	fs vfs.FS
	// bootstrap is a boolean used to indicate whether to bootstrap a new cluster.
	bootstrap bool
	// transport is the transport package for the messages that aspen exchanges.
	// this setting overrides all other transport settings in sub-configs.
	transport Transport
	// externalTransport is a boolean flag indicating whether the caller provided an external
	// transport that they control themselves.
	externalTransport bool
	// logger is the witness of it all.
	logger *zap.SugaredLogger
	// experiment is the experiment that aspen attaches its metrics to.
	experiment alamos.Experiment
}

func (o *options) Report() alamos.Report {
	// The key-value store and cluster state services will attach their own reports to the experiment,
	// so we only need to report values that they won't.
	return alamos.Report{
		"dirname":   o.dirname,
		"addr":      o.addr,
		"peers":     o.peerAddresses,
		"bootstrap": o.bootstrap,
	}
}

// Bootstrap tells aspen to bootstrap a new cluster. This option automatically assigns the host node and NodeID of 1.
func Bootstrap() Option { return func(o *options) { o.bootstrap = true } }

// WithLogger sets the logger for aspen.
func WithLogger(logger *zap.SugaredLogger) Option { return func(o *options) { o.logger = logger } }

// WithExperiment sets the experiment for aspen. Aspen will attach any metrics and reports it generates to this
// experiment.
func WithExperiment(experiment alamos.Experiment) Option {
	return func(o *options) { o.experiment = experiment }
}

// WithEngine sets the underlying KV engine that aspen uses to store its data. When using this option, the caller
// should transfer all responsibility for executing queries on the engine to aspen.
func WithEngine(engine kvx.DB) Option {
	return func(o *options) {
		o.externalKV = true
		o.kv.Engine = engine
	}
}

// MemBacked sets aspen to use a memory-backed KV engine. This option is ignored if a custom KV engine is set (using
// WithEngine).
func MemBacked() Option {
	return func(o *options) {
		o.dirname = ""
		o.fs = vfs.NewMem()
	}
}

// PropagationConfig is a set of configurable values that tune how quickly state converges across the cluster.
// Lower intervals typically bring faster convergence, but also use considerably more network traffic.
type PropagationConfig struct {
	// PledgeRetryInterval is the interval at which aspen will retry sending a pledge to a peer.
	// Pledges are sent at a scaled interval (see PledgeRetryScale).
	PledgeRetryInterval time.Duration
	// PledgeRetryScale is the factory at which the interval increases after failed pledges. For example, a
	// PledgeRetryInterval of 2 seconds and a PledgeRetryScale of 2 will result in pledge intervals of
	// 2, 4, 8, 16, 32, and so on until the pledge is accepted.
	PledgeRetryScale float64
	// PledgeRequestTimeout is the maximum amount of time aspen will wait for a pledge request to be accepted before
	// moving on to the next peer.
	PledgeRequestTimeout time.Duration
	// ClusterGossipInterval is the interval at which aspen will propagate cluster state to other nodes.
	// Aspen will send messages regardless of whether the state has changed, so setting this interval to a low
	// value may result in very high network traffic.
	ClusterGossipInterval time.Duration
	KVGossipInterval      time.Duration
}

// WithPropagationConfig sets the parameters defining how quickly cluster state converges. See PropagationConfig
// for more details.
func WithPropagationConfig(config PropagationConfig) Option {
	return func(o *options) {
		o.cluster.Pledge.RetryInterval = config.PledgeRetryInterval
		o.cluster.Pledge.RetryScale = config.PledgeRetryScale
		o.cluster.Pledge.RequestTimeout = config.PledgeRequestTimeout
		o.cluster.Gossip.Interval = config.ClusterGossipInterval
		o.kv.GossipInterval = config.KVGossipInterval
	}
}

var FastPropagationConfig = PropagationConfig{
	PledgeRetryInterval:   10 * time.Millisecond,
	PledgeRetryScale:      1,
	ClusterGossipInterval: 50 * time.Millisecond,
	KVGossipInterval:      50 * time.Millisecond,
}

// WithTransport sets a custom network transport.
func WithTransport(transport Transport) Option {
	return func(o *options) {
		o.externalTransport = true
		o.transport = transport
	}
}

func newOptions(dirname string, addr address.Address, peers []address.Address, opts ...Option) *options {
	o := &options{}
	o.dirname = dirname
	o.addr = addr
	o.peerAddresses = peers
	for _, opt := range opts {
		opt(o)
	}
	mergeDefaultOptions(o)
	alamos.AttachReporter(o.experiment, "aspen", alamos.Debug, o)
	return o
}

func mergeDefaultOptions(o *options) {
	def := defaultOptions()

	// |||| DIRNAME ||||

	if o.dirname == "" {
		o.dirname = def.dirname
	}

	// |||| KV ||||

	o.kv = def.kv.Override(o.kv)

	// |||| CLUSTER ||||

	o.cluster.Experiment = o.experiment
	o.cluster.Pledge.Peers = o.peerAddresses
	o.cluster.HostAddress = o.addr

	// |||| SHUTDOWN ||||

	// |||| TRANSPORT ||||

	if o.transport == nil {
		o.transport = def.transport
	}

	// |||| LOGGER ||||

	if o.logger == nil {
		o.logger = def.logger
	}
	o.cluster.Logger = o.logger.Named("cluster")
	o.kv.Logger = o.logger.Named("kv")

	if o.bootstrap {
		o.peerAddresses = []address.Address{}
		o.cluster.Pledge.Peers = []address.Address{}
	}

}

func defaultOptions() *options {
	logger, _ := zap.NewProduction()
	return &options{
		dirname:   "",
		cluster:   cluster.DefaultConfig,
		kv:        kv.DefaultConfig,
		transport: grpct.New(fgrpc.NewPool(grpc.WithTransportCredentials(insecure.NewCredentials()))),
		logger:    logger.Sugar(),
	}
}
