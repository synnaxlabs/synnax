// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// LimitChecker returns an error when a count of external channels exceeds the limit.
type LimitChecker = func(types.Uint20) error

// ServiceConfig configures the service-layer channel service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Channel is the distribution-layer channel service the service drives to assign
	// local keys and create, rename, and delete storage channels.
	//
	// [REQUIRED]
	Channel *channel.Service
	// DB is the metadata database backing the channel table.
	//
	// [REQUIRED]
	DB *gorp.DB
	// HostKey is the key of the host node, the default leaseholder of new channels.
	//
	// [REQUIRED]
	HostKey node.Key
	// Limit enforces the cap on external (non-internal, non-virtual) channels.
	//
	// [OPTIONAL] - Defaults to no cap.
	Limit LimitChecker
	// Status publishes error/clear statuses for calculated channels.
	//
	// [REQUIRED]
	Status *status.Service
	// ValidateNames sets whether to validate channel names during creation and
	// renaming.
	//
	// [OPTIONAL] - Defaults to true.
	ValidateNames *bool
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.channel")
	v.NotNil("channel", c.Channel)
	v.NotNil("db", c.DB)
	v.NonZero("host_key", c.HostKey)
	v.NotNil("limit", c.Limit)
	v.NotNil("status", c.Status)
	v.NotNil("validate_names", c.ValidateNames)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.DB = override.Nil(c.DB, other.DB)
	c.HostKey = override.Numeric(c.HostKey, other.HostKey)
	c.Limit = override.Nil(c.Limit, other.Limit)
	c.Status = override.Nil(c.Status, other.Status)
	c.ValidateNames = override.Nil(c.ValidateNames, other.ValidateNames)
	return c
}

// Service is the top-level channel service. It owns the channel metadata table,
// retrieval, search integration, and the create/delete/rename orchestration, driving
// the distribution-layer allocator for key assignment and storage. It also infers
// DataTypes for calculated channels on write.
type Service struct {
	cfg     ServiceConfig
	db      *gorp.DB
	closer  io.MultiCloser
	table   *gorp.Table[Key, Channel]
	indexes indexes
	index   *gorp.KeyReference[Key, Channel, Key]
	state   *actions.State[Key, Action]
	// renames holds the renames of each transaction. The hook of the last one sends all
	// of them to storage in one call.
	renames *gorp.Attachment[rename]
	mu      struct {
		sync.RWMutex
		// externalNonVirtualSet holds the keys of external (non-internal, non-virtual)
		// channels, which count toward the channel limit.
		externalNonVirtualSet set.Integer[Key]
	}
}

// OpenService opens a channel service using the provided configuration(s).
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{
		ValidateNames: new(true),
		Limit:         func(types.Uint20) error { return nil },
	}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		db:      cfg.DB,
		indexes: newIndexes(),
		index:   indexReference(),
		state:   actions.NewState[Key, Action](cfg.DB),
		renames: gorp.NewAttachment[rename](),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Channel]{
		DB:   cfg.DB,
		Name: tableName,
		Migrations: append(
			slices.Clone(versions.Migrations),
			gorp.DanglingMigration("v61_dangling_references"),
		),
		Indexes:         s.indexes.all(),
		References:      append(s.indexes.references(), s.index),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	// Seed the external/non-virtual key set by scanning the table once at startup. This
	// is the only call site that unavoidably walks the table — there is no index for
	// (Internal, Virtual) and the cost is bounded by how many channels the Core has
	// accumulated.
	var externalNonVirtualChannels []Channel
	if err = s.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
			return !c.Internal && !c.Virtual, nil
		})).
		Entries(&externalNonVirtualChannels).
		Exec(ctx, cfg.DB); !ok(err, nil) {
		return nil, err
	}
	s.mu.externalNonVirtualSet.Insert(KeysFromChannels(externalNonVirtualChannels)...)
	return s, nil
}

// newRetrieve returns a Retrieve without the channel limit validator. The create,
// delete, and rename paths use it because they hold the lock validateChannels blocks
// on, and they check the limit at commit time.
func (s *Service) newRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}

// Observe returns an observable that notifies callers of changes to channels.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Channel]] {
	return s.table.Observe()
}

// Changes returns the changes that c made to channels.
func (s *Service) Changes(
	ctx context.Context,
	c gorp.Commit,
) gorp.TxReader[Key, Channel] {
	return s.table.Changes(ctx, c)
}

// NewRetrieve opens a retrieve query that fails when a retrieved external channel is
// past the channel limit.
func (s *Service) NewRetrieve() Retrieve {
	r := s.newRetrieve()
	r.gorp = r.gorp.Validate(s.validateChannels)
	return r
}

// CountLimited returns how many channels count against the license channel limit (not
// internal, not virtual).
func (s *Service) CountLimited() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return uint32(s.mu.externalNonVirtualSet.Size())
}

// Close releases the resources held by the Service.
func (s *Service) Close() error { return s.closer.Close() }

// validateChannels fails a NewRetrieve query when a retrieved external non-virtual
// channel is past the channel limit.
func (s *Service) validateChannels(_ gorp.Context, channels []Channel) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range channels {
		key := ch.Key()
		if !s.mu.externalNonVirtualSet.Contains(key) {
			continue
		}
		channelNumber := s.mu.externalNonVirtualSet.NumLessThan(key) + 1
		if err := s.cfg.Limit(types.Uint20(channelNumber)); err != nil {
			return err
		}
	}
	return nil
}

// NewArcSymbolResolver returns a resolver that maps channels to Arc symbols by name or
// numeric key, for analyzing and compiling Arc expressions such as calculated channels.
// The resolver looks up channels in snap and sees only the channels that every filter
// matches.
func (s *Service) NewArcSymbolResolver(
	snap gorp.Snapshot,
	filters ...Filter,
) arc.SymbolResolver {
	return &symbolResolver{svc: s, snap: snap, filters: filters}
}

// CalculationInputs returns the stored channels that the calculated channels among
// channels read, directly or through other calculated channels, including the channels
// that reset their operations. The channels need not be stored yet. A calculation whose
// expression fails analysis adds nothing, since it cannot run. A failed read returns
// its error. CalculationInputs reads through snap.
func (s *Service) CalculationInputs(
	ctx context.Context,
	snap gorp.Snapshot,
	channels []Channel,
) (Keys, error) {
	analyzer := NewCalculationAnalyzer(s.NewArcSymbolResolver(snap), parser.Config{
		AllowDashedNames: !s.ShouldValidateNames(),
	})
	var (
		inputs  Keys
		visited = set.New[Key]()
	)
	// A write replaces only the stored form of a calculated channel, so the walk still
	// reads the stored form of every other channel among channels.
	for _, ch := range channels {
		if ch.IsCalculated() {
			visited.Add(ch.Key())
		}
	}
	for len(channels) > 0 {
		var next Keys
		for _, ch := range channels {
			if !ch.IsCalculated() {
				continue
			}
			result, err := analyzer.Analyze(ctx, ch)
			if errors.As(err, new(*diagnostics.Diagnostics)) {
				continue
			}
			if err != nil {
				return nil, err
			}
			reads := result.Deps
			for _, op := range ch.Operations {
				if op.ResetChannel != 0 {
					reads = append(reads, op.ResetChannel)
				}
			}
			for _, k := range reads {
				// A channel among channels that is not stored yet has no local key.
				if k.LocalKey() == 0 {
					continue
				}
				inputs = append(inputs, k)
				if !visited.Contains(k) {
					visited.Add(k)
					next = append(next, k)
				}
			}
		}
		channels = nil
		if err := s.NewRetrieve().
			Where(MatchKeys(next...)).
			Where(MatchCalculated()).
			Entries(&channels).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
	}
	return inputs.Unique(), nil
}

// freeIndexResolver adapts the Service to the distribution framer writer's
// FreeIndexResolver interface without exposing resolution on the Service itself.
type freeIndexResolver struct{ svc *Service }

func (r freeIndexResolver) ResolveFreeIndexes(
	ctx context.Context,
	keys Keys,
) (map[Key]Key, error) {
	var channels []Channel
	if err := r.svc.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return r.svc.NewRetrieve().
			Where(MatchKeys(keys...)).
			Entries(&channels).
			Exec(ctx, snap)
	}); err != nil {
		return nil, err
	}
	indexes := make(map[Key]Key)
	for _, ch := range channels {
		if ch.Free() && ch.Index() != 0 {
			indexes[ch.Key()] = ch.Index()
		}
	}
	return indexes, nil
}

// FreeIndexResolver returns the resolver the distribution framer writer uses to stamp
// free frames with per-index alignments.
func (s *Service) FreeIndexResolver() writer.FreeIndexResolver {
	return freeIndexResolver{svc: s}
}

// NewWriter returns a Writer scoped to tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		svc: s,
		tx:  tx,
		analyzer: NewCalculationAnalyzer(s.NewArcSymbolResolver(tx), parser.Config{
			AllowDashedNames: !s.ShouldValidateNames(),
		}),
		dispatcher: s.state.Dispatcher(),
	}
}

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// ShouldValidateNames reports whether channel-name validation is enabled.
func (s *Service) ShouldValidateNames() bool { return *s.cfg.ValidateNames }
