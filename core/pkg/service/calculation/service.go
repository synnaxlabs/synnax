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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
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
	// Framer reads the newest stored samples of the channels that a calculation reads,
	// to pick its primary.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status stores the status that a save posts when it picks a new primary.
	//
	// [REQUIRED]
	Status *status.Service
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
	v.NotNil("framer", c.Framer)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Service stores calculations.
type Service struct {
	closer   xio.MultiCloser
	table    *gorp.Table[Key, Calculation]
	indexes  indexes
	state    *actions.State[Key, Action]
	channels *channel.Service
	framer   *framer.Service
	statuses *status.Service
	db       *gorp.DB
	// checks holds the check of each action batch that a transaction attaches.
	checks *gorp.Attachment[batchCheck[Key]]
	// outputChecks holds the check of each channel action batch that announces the
	// calculation fields of an output.
	outputChecks *gorp.Attachment[batchCheck[channel.Key]]
}

// batchCheck checks one action batch by the calculation as the batch leaves it.
type batchCheck[K comparable] struct {
	id    batchID[K]
	check func(allowed func(resource.ID) bool) bool
}

// batchID names one action batch by the key of its entry and its number.
type batchID[K comparable] struct {
	key K
	seq uint64
}

// checksOf returns the check of each batch in attached, by its ID.
func checksOf[K comparable](
	attached []batchCheck[K],
) map[batchID[K]]func(func(resource.ID) bool) bool {
	byBatch := make(map[batchID[K]]func(func(resource.ID) bool) bool, len(attached))
	for _, b := range attached {
		byBatch[b.id] = b.check
	}
	return byBatch
}

// OpenService opens a Service. When the error is nil, the caller must close the
// service.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		indexes:      newIndexes(),
		state:        actions.NewState[Key, Action](cfg.DB),
		channels:     cfg.Channel,
		framer:       cfg.Framer,
		statuses:     cfg.Status,
		db:           cfg.DB,
		checks:       gorp.NewAttachment[batchCheck[Key]](),
		outputChecks: gorp.NewAttachment[batchCheck[channel.Key]](),
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
	held, err := s.indexes.output.Get(ctx, snap, keys...)
	q := s.NewRetrieve()
	switch {
	case errors.Is(err, gorp.ErrIndexInvalid):
		q = q.Where(MatchOutputs(keys...))
	case err != nil:
		return nil, err
	case len(held) == 0:
		return nil, nil
	default:
		// A key from a commit in flight may have no entry in snap.
		q = q.Where(MatchKeys(held...)).SkipMissingKeys()
	}
	var calcs []Calculation
	if err = q.Entries(&calcs).Exec(ctx, snap); err != nil {
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

// Denied returns the first resource that a subject must retrieve to read c and that
// allowed refuses, and false when allowed admits them all. The resources are c and
// each channel that c links. The retrieve and the change channels of calculations both
// decide by it.
func (s *Service) Denied(
	c Calculation,
	allowed func(resource.ID) bool,
) (resource.ID, bool) {
	return s.denied(c, nil, allowed)
}

// Readable returns whether a subject reads c, given whether it can retrieve each
// resource. It finds the channels of c once, so a check of many subjects reads them
// once.
func (s *Service) Readable(c Calculation) func(allowed func(resource.ID) bool) bool {
	channels := s.Channels(c)
	return func(allowed func(resource.ID) bool) bool {
		_, refused := s.denied(c, channels, allowed)
		return !refused
	}
}

// denied implements Denied. channels holds the channels that c links when known, and
// nil makes denied find them only when allowed admits c.
func (s *Service) denied(
	c Calculation,
	channels []resource.ID,
	allowed func(resource.ID) bool,
) (resource.ID, bool) {
	calc := ResourceID(c.Key)
	if channels == nil {
		if !allowed(calc) {
			return calc, true
		}
		channels = s.Channels(c)
	}
	return refused(calc, channels, allowed)
}

// refused returns the first that allowed refuses of calc, the resource of a
// calculation, and channels, the channels that the calculation links.
func refused(
	calc resource.ID,
	channels []resource.ID,
	allowed func(resource.ID) bool,
) (resource.ID, bool) {
	if !allowed(calc) {
		return calc, true
	}
	for _, id := range channels {
		if !allowed(id) {
			return id, true
		}
	}
	return resource.ID{}, false
}

// BatchChecks returns the lookup of the check of each action batch that the transaction
// of c attached, by the key of its calculation and its number. Each check is Readable
// of the calculation as its batch left it, so a later batch in the commit does not
// change who receives an earlier one.
func (s *Service) BatchChecks(
	c gorp.Commit,
) func(key Key, seq uint64) (func(allowed func(resource.ID) bool) bool, bool) {
	byBatch := checksOf(s.checks.Of(c))
	return func(key Key, seq uint64) (func(func(resource.ID) bool) bool, bool) {
		check, ok := byBatch[batchID[Key]{key: key, seq: seq}]
		return check, ok
	}
}

// DeleteChecks returns the lookup of the check of each calculation that c deleted, by
// key. Only a subject that read the calculation before the delete passes it.
func (s *Service) DeleteChecks(
	c gorp.Commit,
) func(key Key) (func(allowed func(resource.ID) bool) bool, bool) {
	byKey := make(map[string]func(func(resource.ID) bool) bool)
	for _, ch := range c.Changes {
		if ch.Record.Table != tableName || ch.Kind != gorp.ChangeDelete {
			continue
		}
		calc := resource.ID{Type: resource.TypeCalculation, Key: ch.Record.Key}
		channels := make([]resource.ID, len(ch.Removed))
		for i, l := range ch.Removed {
			channels[i] = resource.ID{
				Type: resource.Type(l.Target.Table),
				Key:  l.Target.Key,
			}
		}
		byKey[ch.Record.Key] = func(allowed func(resource.ID) bool) bool {
			_, denied := refused(calc, channels, allowed)
			return !denied
		}
	}
	return func(key Key) (func(func(resource.ID) bool) bool, bool) {
		check, ok := byKey[gorp.FormatKey(key)]
		return check, ok
	}
}

// NewWriter opens a Writer that changes calculations in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		svc:        s,
		tx:         tx,
		table:      s.table,
		indexes:    s.indexes,
		channels:   s.channels,
		dispatcher: s.state.Dispatcher(),
	}
}
