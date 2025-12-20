// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// All included pebble code is copyrighted by the cockroachdb team, and is licensed under
// the BSD 3-Clause License. See the repository file license/BSD-3-Clause.txt for more
// information.

package aspen

import (
	"time"

	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	grpct "github.com/synnaxlabs/aspen/transport/grpc"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/x/address"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Option is a function that configures an Aspen instance.
type Option func(*options)

type options struct {
	alamos.Instrumentation
	// dirname is the directory where aspen will store its data.
	// this option is ignored if a custom kv.ServiceConfig.Engine is set.
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
	// if a custom kv.ServiceConfig.Engine is set.
	fs vfs.FS
	// bootstrap is a boolean used to indicate whether to bootstrap a new cluster.
	bootstrap bool
	// transport is the transport package for the messages that aspen exchanges.
	// this setting overrides all other transport settings in sub-configs.
	transport struct {
		Transport
		// external is a boolean flag indicating whether the caller provided an
		// external transport they control themselves.
		external bool
	}
}

func (o *options) Report() alamos.Report {
	// The key-value store and cluster state services will attach their own reports to
	// the instrumentation, so we only need to report values that they won't.
	return alamos.Report{
		"dirname":   o.dirname,
		"addr":      o.addr,
		"peers":     o.peerAddresses,
		"bootstrap": o.bootstrap,
	}
}

// Bootstrap tells aspen to bootstrap a new cluster. This option automatically assigns
// the host node and NodeID of 1.
func Bootstrap() Option { return func(o *options) { o.bootstrap = true } }

// WithEngine sets the underlying KV engine that aspen uses to store its data. When
// using this option, the caller should transfer all responsibility for executing queries
// on the engine to aspen.
func WithEngine(engine xkv.DB) Option {
	return func(o *options) {
		o.externalKV = true
		o.kv.Engine = engine
	}
}

// WithTransport sets a custom network transport.
func WithTransport(transport Transport) Option {
	return func(o *options) {
		o.transport.external = true
		o.transport.Transport = transport
	}
}

// WithInstrumentation sets the instrumentation for aspen.
func WithInstrumentation(i alamos.Instrumentation) Option {
	return func(o *options) {
		o.Instrumentation = i
	}
}

// InMemory sets aspen to use a memory-backed KV engine. This option is ignored if a
// custom KV engine is set (using WithEngine).
func InMemory() Option {
	return func(o *options) { o.dirname = ""; o.fs = vfs.NewMem() }
}

// PropagationConfig is a set of configurable values that tune how quickly state converges
// across the cluster. Lower intervals typically bring faster convergence, but also use
// considerably more network traffic.
type PropagationConfig struct {
	// PledgeRetryInterval is the interval at which aspen will retry sending a pledge to
	// a peer. Pledges are sent at a scaled interval (see PledgeRetryScale).
	PledgeRetryInterval time.Duration
	// PledgeRetryScale is the factory at which the interval increases after failed
	// pledges. For example, a PledgeRetryInterval of 2 seconds and a PledgeRetryScale
	// of 2 will result in pledge intervals of 2, 4, 8, 16, 32, and so on until the
	// pledge is accepted.
	PledgeRetryScale float64
	// PledgeRequestTimeout is the maximum amount of time aspen will wait for a pledge
	// request to be accepted before moving on to the next peer.
	PledgeRequestTimeout time.Duration
	// ClusterGossipInterval is the interval at which aspen will propagate cluster state
	// to other nodes. Aspen will send messages regardless of whether the state has
	// changed, so setting this interval to a low value may result in very high network
	// traffic.
	ClusterGossipInterval time.Duration
	// KVGossipInterval sets the interval at which aspen will propagate key-Value
	// operations to other nodes. It's important to note that KV will not gossip if
	// there are no operations to propagate.
	KVGossipInterval time.Duration
}

// WithPropagationConfig sets the parameters defining how quickly cluster state converges.
// See PropagationConfig for more details.
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
	ClusterGossipInterval: 10 * time.Millisecond,
	KVGossipInterval:      10 * time.Millisecond,
}

func newOptions(
	dirname string,
	addr address.Address,
	peers []address.Address,
	opts ...Option,
) *options {
	o := &options{
		dirname:       dirname,
		addr:          addr,
		peerAddresses: peers,
	}
	for _, opt := range opts {
		opt(o)
	}
	mergeDefaultOptions(o)
	return o
}

func mergeDefaultOptions(o *options) {
	def := defaultOptions()
	o.dirname = override.String(def.dirname, o.dirname)
	o.kv = def.kv.Override(o.kv)
	o.cluster = def.cluster.Override(o.cluster)
	o.transport.Transport = override.Nil(def.transport.Transport, o.transport.Transport)
	o.Instrumentation = override.Zero(def.Instrumentation, o.Instrumentation)
	o.cluster.Instrumentation = o.Child("cluster")
	o.kv.Instrumentation = o.Child("kv")
	o.cluster.HostAddress = o.addr
	o.cluster.Pledge.Peers = o.peerAddresses
	// If we're bootstrapping these options are ignored.
	if o.bootstrap {
		o.peerAddresses = []address.Address{}
		o.cluster.Pledge.Peers = []address.Address{}
	}
}

func defaultOptions() *options {
	o := &options{
		dirname: "aspen",
		cluster: cluster.DefaultConfig,
		kv:      kv.DefaultConfig,
	}
	o.transport.Transport = grpct.New(
		fgrpc.NewPool("", grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	return o
}
