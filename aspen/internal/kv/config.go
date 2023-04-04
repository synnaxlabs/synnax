// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"time"
)

// Config is the configuration for the aspen db service. For default values, see DefaultConfig().
type Config struct {
	alamos.Instrumentation
	// Cluster is the cluster that the DB will use to communicate with other databases.
	// [Required]
	Cluster cluster.Cluster
	// BatchTransportClient is used to send key-value Operations to nodes.
	// [Required]
	BatchTransportClient BatchTransportClient
	// BatchTransportServer is used to receive key-value Operations from nodes.
	// [Required]
	BatchTransportServer BatchTransportServer
	// FeedbackTransportClient is used to send gossip feedback to nodes.
	// [Required]
	FeedbackTransportClient FeedbackTransportClient
	// FeedbackTransportServer is used to receive gossip feedback from nodes.
	// [Required]
	FeedbackTransportServer FeedbackTransportServer
	// LeaseTransportClient is used to receive leaseAlloc Operations between nodes.
	// [Required]
	LeaseTransportClient LeaseTransportClient
	// LeaseTransportServer is used to send leaseAlloc Operations between nodes.
	// [Required]
	LeaseTransportServer LeaseTransportServer
	// Engine is the underlying key-value engine that DB writes its Operations to.
	// [Required]
	Engine kvx.DB
	// GossipInterval is how often a node initiates gossip with a peer.
	// [Not Required]
	GossipInterval time.Duration
	// Recovery threshold for the SIR gossip protocol i.e. how many times the node must send a redundant operation
	// for it to stop propagating it.
	// [Not Required]
	RecoveryThreshold int
}

func (cfg Config) Override(other Config) Config {
	cfg.Cluster = override.Nil(cfg.Cluster, other.Cluster)
	cfg.BatchTransportClient = override.Nil(cfg.BatchTransportClient, other.BatchTransportClient)
	cfg.BatchTransportServer = override.Nil(cfg.BatchTransportServer, other.BatchTransportServer)
	cfg.FeedbackTransportClient = override.Nil(cfg.FeedbackTransportClient, other.FeedbackTransportClient)
	cfg.FeedbackTransportServer = override.Nil(cfg.FeedbackTransportServer, other.FeedbackTransportServer)
	cfg.LeaseTransportServer = override.Nil(cfg.LeaseTransportServer, other.LeaseTransportServer)
	cfg.LeaseTransportClient = override.Nil(cfg.LeaseTransportClient, other.LeaseTransportClient)
	cfg.Engine = override.Nil(cfg.Engine, other.Engine)
	cfg.GossipInterval = override.Numeric(cfg.GossipInterval, other.GossipInterval)
	cfg.RecoveryThreshold = override.Numeric(cfg.RecoveryThreshold, other.RecoveryThreshold)
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("db")
	validate.NotNil(v, "cluster", cfg.Cluster)
	validate.NotNil(v, "BatchTransportClient", cfg.BatchTransportClient)
	validate.NotNil(v, "BatchTransportServer", cfg.BatchTransportServer)
	validate.NotNil(v, "FeedbackTransportClient", cfg.FeedbackTransportClient)
	validate.NotNil(v, "FeedbackTransportServer", cfg.FeedbackTransportServer)
	validate.NotNil(v, "LeaseTransportClient", cfg.LeaseTransportServer)
	validate.NotNil(v, "LeaseTransportServer", cfg.LeaseTransportClient)
	validate.NotNil(v, "Engine", cfg.Engine)
	return v.Error()
}

func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	report["recoveryThreshold"] = cfg.RecoveryThreshold
	report["gossipInterval"] = cfg.GossipInterval.String()
	report["batchTransportClient"] = cfg.BatchTransportClient.Report()
	report["batchTransportServer"] = cfg.BatchTransportServer.Report()
	report["feedbackTransportClient"] = cfg.FeedbackTransportClient.Report()
	report["feedbackTransportServer"] = cfg.FeedbackTransportServer.Report()
	report["leaseTransportClient"] = cfg.LeaseTransportClient.Report()
	report["leaseTransportServer"] = cfg.LeaseTransportServer.Report()
	return report
}

var DefaultConfig = Config{
	GossipInterval:    1 * time.Second,
	RecoveryThreshold: 5,
}
