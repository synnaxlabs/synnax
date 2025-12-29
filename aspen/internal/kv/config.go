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
	xkv "github.com/synnaxlabs/x/kv"
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
	Engine xkv.DB
	// GossipInterval is how often a node initiates gossip with a peer.
	// [Not Required]
	GossipInterval time.Duration
	// Recovery threshold for the SIR gossip protocol, i.e., how many times the node
	// must send a redundant operation for it to stop propagating it.
	// [Not Required]
	RecoveryThreshold int
}

// Override implements config.Config.
func (cfg Config) Override(other Config) Config {
	cfg.Cluster = override.Nil(cfg.Cluster, other.Cluster)
	cfg.BatchTransportClient = override.Nil(cfg.BatchTransportClient, other.BatchTransportClient)
	cfg.BatchTransportServer = override.Nil(cfg.BatchTransportServer, other.BatchTransportServer)
	cfg.FeedbackTransportClient = override.Nil(cfg.FeedbackTransportClient, other.FeedbackTransportClient)
	cfg.FeedbackTransportServer = override.Nil(cfg.FeedbackTransportServer, other.FeedbackTransportServer)
	cfg.LeaseTransportServer = override.Nil(cfg.LeaseTransportServer, other.LeaseTransportServer)
	cfg.LeaseTransportClient = override.Nil(cfg.LeaseTransportClient, other.LeaseTransportClient)
	cfg.RecoveryTransportClient = override.Nil(cfg.RecoveryTransportClient, other.RecoveryTransportClient)
	cfg.RecoveryTransportServer = override.Nil(cfg.RecoveryTransportServer, other.RecoveryTransportServer)
	cfg.Engine = override.Nil(cfg.Engine, other.Engine)
	cfg.GossipInterval = override.Numeric(cfg.GossipInterval, other.GossipInterval)
	cfg.RecoveryThreshold = override.Numeric(cfg.RecoveryThreshold, other.RecoveryThreshold)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("cesium")
	validate.NotNil(v, "cluster", cfg.Cluster)
	validate.NotNil(v, "tx_transport_client", cfg.BatchTransportClient)
	validate.NotNil(v, "tx_transport_server", cfg.BatchTransportServer)
	validate.NotNil(v, "feedback_transport_client", cfg.FeedbackTransportClient)
	validate.NotNil(v, "feedback_transport_server", cfg.FeedbackTransportServer)
	validate.NotNil(v, "lease_transport_client", cfg.LeaseTransportServer)
	validate.NotNil(v, "lease_transport_server", cfg.LeaseTransportClient)
	validate.NotNil(v, "recovery_transport_client", cfg.RecoveryTransportClient)
	validate.NotNil(v, "recovery_transport_server", cfg.RecoveryTransportServer)
	validate.NotNil(v, "engine", cfg.Engine)
	return v.Error()
}

// Report implements alamos.ReportProvider.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["recovery_threshold"] = cfg.RecoveryThreshold
	report["gossip_interval"] = cfg.GossipInterval.String()
	report["batch_transport_client"] = cfg.BatchTransportClient.Report()
	report["batch_transport_server"] = cfg.BatchTransportServer.Report()
	report["feedback_transport_client"] = cfg.FeedbackTransportClient.Report()
	report["feedback_transport_server"] = cfg.FeedbackTransportServer.Report()
	report["lease_transport_client"] = cfg.LeaseTransportClient.Report()
	report["lease_transport_server"] = cfg.LeaseTransportServer.Report()
	return report
}

// DefaultConfig is the default configuration for the key-value service.
var DefaultConfig = Config{
	GossipInterval:    1 * time.Second,
	RecoveryThreshold: 5,
}
