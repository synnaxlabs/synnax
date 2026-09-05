// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pledge

import (
	"time"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	// Request is the payload exchanged during a pledge. A zero Key asks the receiving
	// peer to act as responsible for the pledge; a non-zero Key proposes that key to
	// the receiving juror.
	Request struct {
		// Key is the proposed node key, or zero for an initial pledge request.
		Key node.Key
		// ClusterKey is the unique key of the cluster the node is pledging to.
		ClusterKey uuid.UUID
	}
	// Response is returned to a successful pledge, carrying the assigned node key and
	// the key of the cluster the node was inducted into.
	Response = Request
	// TransportClient sends pledge requests and proposals to peers.
	TransportClient = freighter.UnaryClient[Request, Response]
	// TransportServer receives pledge requests and proposals from peers.
	TransportServer = freighter.UnaryServer[Request, Response]
)

// Config is used for configuring a pledge based membership network.
type Config struct {
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// TransportClient is used for sending pledge information over the network.
	//
	// [REQUIRED]
	TransportClient TransportClient
	// TransportServer is used for receiving pledge information over the network.
	//
	// [REQUIRED]
	TransportServer TransportServer
	// Candidates is a group of nodes to contact as candidates for the formation
	// of a jury.
	//
	// [REQUIRED]
	Candidates func() node.Group
	// Peers is a set of addresses a pledge can contact.
	//
	// [OPTIONAL] - Required when pledging.
	Peers []address.Address
	// RequestTimeout is the timeout for a peer to respond to a pledge or proposal
	// request. If the request is not responded to before the timeout, a new jury will
	// be formed and the request will be retried.
	//
	// [OPTIONAL] - Defaults to 5 seconds.
	RequestTimeout time.Duration
	// RetryInterval sets the initial retry interval for a Pledge to a peer.
	//
	// [OPTIONAL] - Defaults to 1 second.
	RetryInterval time.Duration
	// MaxProposals is the maximum number of failed quorum consultations a responsible
	// node will tolerate before giving up. Proposals rejected by a juror retry with a
	// higher key and do not count against this limit, so the total number of rounds is
	// bounded only by cancellation of the pledge request.
	//
	// [OPTIONAL] - Defaults to 10.
	MaxProposals uint
	// RetryScale sets how quickly the time in-between retries will increase during a
	// Pledge to a peer. For example, a value of 2 would result in a retry interval of
	// 1, 2, 4, 8, 16, 32, 64, ... seconds.
	//
	// [OPTIONAL] - Defaults to 1.25.
	RetryScale float64
	// ClusterKey is a unique key for the cluster. This value is consistent across all
	// nodes in the cluster.
	//
	// [OPTIONAL] - Required when arbitrating. A pledging node learns the key from
	// the pledge response.
	ClusterKey uuid.UUID
}

var _ config.Config[Config] = Config{}

// Override implements the config.Config interface.
func (cfg Config) Override(other Config) Config {
	cfg.TransportClient = override.Nil(cfg.TransportClient, other.TransportClient)
	cfg.TransportServer = override.Nil(cfg.TransportServer, other.TransportServer)
	cfg.ClusterKey = override.If(
		cfg.ClusterKey,
		other.ClusterKey,
		other.ClusterKey != uuid.Nil,
	)
	cfg.RequestTimeout = override.Numeric(cfg.RequestTimeout, other.RequestTimeout)
	cfg.RetryInterval = override.Numeric(cfg.RetryInterval, other.RetryInterval)
	cfg.RetryScale = override.Numeric(cfg.RetryScale, other.RetryScale)
	cfg.MaxProposals = override.Numeric(cfg.MaxProposals, other.MaxProposals)
	cfg.Candidates = override.Nil(cfg.Candidates, other.Candidates)
	cfg.Peers = override.Slice(cfg.Peers, other.Peers)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// Validate implements the config.Config interface.
func (cfg Config) Validate() error {
	v := validate.New("pledge")
	v.NotNil("transport_client", cfg.TransportClient)
	v.NotNil("transport_server", cfg.TransportServer)
	v.Positive("request_timeout", cfg.RequestTimeout)
	v.Positive("retry_interval", cfg.RetryInterval)
	v.GreaterThanEq("retry_scale", cfg.RetryScale, 1)
	v.Positive("max_proposals", cfg.MaxProposals)
	v.NotNil("candidates", cfg.Candidates)
	return v.Error()
}

// validatePeers validates the fields Validate cannot, as arbitrating nodes share the
// same Config and need no peers.
func (cfg Config) validatePeers() error {
	v := validate.New("pledge")
	v.NotEmptySlice("peers", cfg.Peers)
	return v.Error()
}

// Report implements the alamos.ReportProvider interface.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["cluster_key"] = cfg.ClusterKey.String()
	report["transport_client"] = cfg.TransportClient.Report()
	report["transport_server"] = cfg.TransportServer.Report()
	report["request_timeout"] = cfg.RequestTimeout
	report["retry_interval"] = cfg.RetryInterval
	report["retry_scale"] = cfg.RetryScale
	report["max_proposals"] = cfg.MaxProposals
	report["peers"] = cfg.Peers
	return report
}

var (
	// DefaultConfig is the default configuration for pledging and arbitrating.
	DefaultConfig = Config{
		RequestTimeout: 5 * time.Second,
		RetryInterval:  1 * time.Second,
		RetryScale:     1.25,
		MaxProposals:   10,
	}
	// FastConfig is DefaultConfig with shortened timeouts and retry intervals for rapid
	// pledges on low-latency networks.
	FastConfig = DefaultConfig.Override(Config{
		RequestTimeout: 50 * time.Millisecond,
		RetryInterval:  10 * time.Millisecond,
		RetryScale:     1.125,
	})
	// BlazingFastConfig is DefaultConfig with near-instant timeouts and retry intervals
	// for in-memory transports in tests.
	BlazingFastConfig = DefaultConfig.Override(Config{
		RequestTimeout: 5 * time.Millisecond,
		RetryInterval:  1 * time.Microsecond,
	})
)
