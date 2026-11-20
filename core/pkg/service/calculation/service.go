// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// DB stores the calculations.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channel retrieves the output of each calculation and the records that use it.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("calculation")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

// Service stores calculations.
type Service struct {
	closer   xio.MultiCloser
	table    *gorp.Table[Key, Calculation]
	indexes  indexes
	state    *actions.State[Key, Action]
	channels *channel.Service
}

// OpenService opens a Service. When the error is nil, the caller must close the
// service.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		indexes:  newIndexes(),
		state:    actions.NewState[Key, Action](cfg.DB),
		channels: cfg.Channel,
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Calculation]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	return s, nil
}

// Close releases the resources of the service.
func (s *Service) Close() error { return s.closer.Close() }

// Observe returns an observable that fires with the changes each commit makes to
// calculations.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Calculation]] {
	return s.table.Observe()
}

// Changes returns the changes that c made to calculations.
func (s *Service) Changes(c gorp.Commit) gorp.TxReader[Key, Calculation] {
	return s.table.Changes(c)
}

// Actions returns the action batches that writers attach to their transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// NewRetrieve opens a query for calculations.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}

// Outputs returns the calculation of each channel among keys that a calculation writes,
// keyed by that channel, as snap sees them.
func (s *Service) Outputs(
	ctx context.Context,
	snap gorp.Snapshot,
	keys ...channel.Key,
) (map[channel.Key]Calculation, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	var calcs []Calculation
	if err := s.NewRetrieve().
		Where(MatchOutputs(keys...)).
		Entries(&calcs).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	byOutput := make(map[channel.Key]Calculation, len(calcs))
	for _, c := range calcs {
		byOutput[c.Output] = c
	}
	return byOutput, nil
}

// Channels returns the resource of each channel that c links, in the order of its
// fields.
func (s *Service) Channels(c Calculation) []resource.ID {
	targets := s.table.Targets(&c)
	ids := make([]resource.ID, len(targets))
	for i, t := range targets {
		ids[i] = resource.ID{Type: resource.Type(t.Table), Key: t.Key}
	}
	return ids
}

// NewWriter opens a Writer that changes calculations in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		indexes:    s.indexes,
		channels:   s.channels,
		dispatcher: s.state.Dispatcher(),
	}
}
