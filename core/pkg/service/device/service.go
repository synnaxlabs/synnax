// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"
	"io"

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for creating a device service.
type Config struct {
	// DB is the gorp database that devices will be stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between devices and other resources in
	// the Synnax cluster.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create device related groups of ontology resources.
	// [REQUIRED]
	Group *group.Service
	// Status is used to define and process statuses for devices.
	// [REQUIRED]
	Status *status.Service
	// Signals is used to propagate device changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
}

var _ config.Config[Config] = Config{}

// Override overrides parts of c with the valid parts of other.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate determines whether the configuration can be used for creating a device
// service.
func (c Config) Validate() error {
	v := validate.New("hardware.device")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

var DefaultConfig = Config{}

// Service is the main entrypoint for managing devices within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting devices. It also
// provides mechanisms for listening to changes in devices.
type Service struct {
	cfg             Config
	shutdownSignals io.Closer
	group           group.Group
}

// OpenService opens a new device service using the provided configuration. If error
// is nil, the service is ready for use and must be closed by calling Close to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, "Devices", ontology.RootID)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals == nil {
		return s, nil
	}
	if s.shutdownSignals, err = signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigString[Device](cfg.DB),
	); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the device service and releases any resources that it may have acquired.
func (s *Service) Close() error {
	if s.shutdownSignals == nil {
		return nil
	}
	return s.shutdownSignals.Close()
}

// RootGroup returns the permanent group for devices. Note that racks will be children
// of this group, and devices will be children of racks (or children of groups that are
// children of racks).
func (s *Service) RootGroup() group.Group { return s.group }

// NewWriter opens a new Writer to create, update, and delete devices. If tx is not nil,
// the writer will use that transaction. If tx is nil, the writer will execute
// operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:     tx,
		otg:    s.cfg.Ontology.NewWriter(tx),
		group:  s.group,
		status: status.NewWriter[StatusDetails](s.cfg.Status, tx),
	}
}

// NewRetrieve opens a new Retrieve query to fetch devices from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:    s.cfg.Ontology,
		baseTX: s.cfg.DB,
		gorp:   gorp.NewRetrieve[string, Device](),
	}
}
