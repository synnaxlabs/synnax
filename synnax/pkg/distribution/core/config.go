// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	alamos.Instrumentation
	// AdvertiseAddress is the address the distribution layer will advertise to the rest of the nodes in the Cluster.
	AdvertiseAddress address.Address
	// PeerAddresses is a list of addresses of other nodes to contact in the Cluster for bootstrapping.
	// If no addresses are provided and storage is empty, the distribution layer will bootstrap a new Cluster.
	// If a Cluster already exists in storage, the addresses in this list will be ignored.
	PeerAddresses []address.Address
	// Pool is a pool for grpc connections to other nodes in the Cluster.
	Pool *fgrpc.Pool
	// Storage is the storage configuration to use for the node.
	Storage storage.Config
	// Transports is a list of transports the distribution uses for communication.
	// These Transports must be bound to the node's grpc server.
	Transports *[]fgrpc.BindableTransport
}

var _ config.Config[Config] = Config{}

// Override implements Config.
func (cfg Config) Override(other Config) Config {
	cfg.AdvertiseAddress = override.String(cfg.AdvertiseAddress, other.AdvertiseAddress)
	cfg.PeerAddresses = override.Slice(cfg.PeerAddresses, other.PeerAddresses)
	cfg.Pool = override.Nil(cfg.Pool, other.Pool)
	cfg.Transports = override.Nil(cfg.Transports, other.Transports)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.Storage = cfg.Storage.Override(other.Storage)
	return cfg
}

// Validate implements Config.
func (cfg Config) Validate() error {
	v := validate.New("distribution.core")
	return v.Error()
}

var DefaultConfig = Config{
	Storage: storage.DefaultConfig,
}
