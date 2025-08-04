// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for the aspen DB service. For default values, see DefaultConfig().
type Config struct {
	alamos.Instrumentation
	// Cluster is the cluster that the DB will use to communicate with other databases.
	// [Required]
	Cluster *cluster.Cluster
	// BatchTransportClient is used to send key-value NewStreamer to nodes.
	// [Required]
	BatchTransportClient TxTransportClient
	// BatchTransportServer is used to receive key-value NewStreamer from nodes.
	// [Required]
	BatchTransportServer TxTransportServer
	// FeedbackTransportClient is used to send gossip feedback to nodes.
	// [Required]
	FeedbackTransportClient FeedbackTransportClient
	// FeedbackTransportServer is used to receive gossip feedback from nodes.
	// [Required]
	FeedbackTransportServer FeedbackTransportServer
	// LeaseTransportClient is used to receive leaseAlloc NewStreamer between nodes.
	// [Required]
	LeaseTransportClient LeaseTransportClient
	// LeaseTransportServer is used to send leaseAlloc NewStreamer between nodes.
	// [Required]
	LeaseTransportServer LeaseTransportServer
	// RecoveryTransportClient is used to send recovery requests to nodes.
	// [Required]
	RecoveryTransportClient RecoveryTransportClient
	// RecoveryTransportServer is used to receive recovery requests from nodes.
	// [Required]
	RecoveryTransportServer RecoveryTransportServer
	// Engine is the underlying key-value engine that DB writes its key-value pairs to.
	// [Required]
	Engine kv.DB
	// GossipInterval is how often a node initiates gossip with a peer.
	// [Not Required]
	GossipInterval time.Duration
	// Recovery threshold for the SIR gossip protocol, i.e., how many times the node
	// must send a redundant operation for it to stop propagating it.
	// [Not Required]
	RecoveryThreshold int
}

var (
	_ config.Config[Config] = Config{}
	_ alamos.ReportProvider = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	c.BatchTransportClient = override.Nil(c.BatchTransportClient, other.BatchTransportClient)
	c.BatchTransportServer = override.Nil(c.BatchTransportServer, other.BatchTransportServer)
	c.FeedbackTransportClient = override.Nil(c.FeedbackTransportClient, other.FeedbackTransportClient)
	c.FeedbackTransportServer = override.Nil(c.FeedbackTransportServer, other.FeedbackTransportServer)
	c.LeaseTransportServer = override.Nil(c.LeaseTransportServer, other.LeaseTransportServer)
	c.LeaseTransportClient = override.Nil(c.LeaseTransportClient, other.LeaseTransportClient)
	c.RecoveryTransportClient = override.Nil(c.RecoveryTransportClient, other.RecoveryTransportClient)
	c.RecoveryTransportServer = override.Nil(c.RecoveryTransportServer, other.RecoveryTransportServer)
	c.Engine = override.Nil(c.Engine, other.Engine)
	c.GossipInterval = override.Numeric(c.GossipInterval, other.GossipInterval)
	c.RecoveryThreshold = override.Numeric(c.RecoveryThreshold, other.RecoveryThreshold)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("cesium")
	validate.NotNil(v, "cluster", c.Cluster)
	validate.NotNil(v, "tx_transport_client", c.BatchTransportClient)
	validate.NotNil(v, "tx_transport_server", c.BatchTransportServer)
	validate.NotNil(v, "feedback_transport_client", c.FeedbackTransportClient)
	validate.NotNil(v, "feedback_transport_server", c.FeedbackTransportServer)
	validate.NotNil(v, "lease_transport_client", c.LeaseTransportServer)
	validate.NotNil(v, "lease_transport_server", c.LeaseTransportClient)
	validate.NotNil(v, "recovery_transport_client", c.RecoveryTransportClient)
	validate.NotNil(v, "recovery_transport_server", c.RecoveryTransportServer)
	validate.NotNil(v, "engine", c.Engine)
	return v.Error()
}

// Report implements alamos.ReportProvider.
func (c Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["recovery_threshold"] = c.RecoveryThreshold
	report["gossip_interval"] = c.GossipInterval.String()
	report["batch_transport_client"] = c.BatchTransportClient.Report()
	report["batch_transport_server"] = c.BatchTransportServer.Report()
	report["feedback_transport_client"] = c.FeedbackTransportClient.Report()
	report["feedback_transport_server"] = c.FeedbackTransportServer.Report()
	report["lease_transport_client"] = c.LeaseTransportClient.Report()
	report["lease_transport_server"] = c.LeaseTransportServer.Report()
	return report
}

// DefaultConfig is the default configuration for the key-value service.
var DefaultConfig = Config{
	GossipInterval:    1 * time.Second,
	RecoveryThreshold: 5,
}
