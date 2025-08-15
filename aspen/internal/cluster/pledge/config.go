// Copyright 2025 Synnax Labs, Inc.
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
	Request struct {
		Key        node.Key
		ClusterKey uuid.UUID
	}
	Response        = Request
	TransportClient = freighter.UnaryClient[Request, Response]
	TransportServer = freighter.UnaryServer[Request, Response]
)

// Config is used for configuring a pledge based membership network. It implements
// the config.ServiceConfig interface.
type Config struct {
	alamos.Instrumentation
	// Candidates is a group of nodes to contact as candidates for the formation
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
	// ClusterKey is a unique key for the cluster. This value is consistent across
	// all nodes in the cluster.
	// [Required]
	ClusterKey uuid.UUID
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
}

var _ config.Config[Config] = Config{}

// Override implements the config.ServiceConfig interface.
func (c Config) Override(other Config) Config {
	c.TransportClient = override.Nil(c.TransportClient, other.TransportClient)
	c.TransportServer = override.Nil(c.TransportServer, other.TransportServer)
	c.ClusterKey = override.If(c.ClusterKey, other.ClusterKey, other.ClusterKey != uuid.Nil)
	c.RequestTimeout = override.Numeric(c.RequestTimeout, other.RequestTimeout)
	c.RetryInterval = override.Numeric(c.RetryInterval, other.RetryInterval)
	c.RetryScale = override.Numeric(c.RetryScale, other.RetryScale)
	c.MaxProposals = override.Numeric(c.MaxProposals, other.MaxProposals)
	c.Candidates = override.Nil(c.Candidates, other.Candidates)
	c.Peers = override.Slice(c.Peers, other.Peers)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements the config.ServiceConfig interface.
func (c Config) Validate() error {
	v := validate.New("pledge")
	validate.NotNil(v, "transport_client", c.TransportClient)
	validate.NotNil(v, "transport_server", c.TransportServer)
	validate.Positive(v, "request_timeout", c.RequestTimeout)
	validate.GreaterThanEq(v, "retry_scale", c.RetryScale, 1)
	validate.NonZero(v, "max_proposals", c.MaxProposals)
	validate.NotNil(v, "candidates", c.Candidates)
	return v.Error()
}

// Report implements the alamos.ReportProvider interface. Assumes the Config is valid.
func (c Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["cluster_key"] = c.ClusterKey.String()
	report["transport_client"] = c.TransportClient.Report()
	report["transport_server"] = c.TransportServer.Report()
	report["request_timeout"] = c.RequestTimeout
	report["pledge_retry_interval"] = c.RetryInterval
	report["pledge_retry_scale"] = c.RetryScale
	report["max_proposals"] = c.MaxProposals
	report["peers"] = c.Peers
	return report
}

var (
	DefaultConfig = Config{
		RequestTimeout: 5 * time.Second,
		RetryInterval:  1 * time.Second,
		RetryScale:     1.25,
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
