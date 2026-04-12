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
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// Service is the central entity for managing channels within Synnax's distribution
// layer. It provides facilities for creating, retrieving, renaming, and deleting
// channels, and owns the cluster-wide lease routing logic previously held by a
// separate leaseProxy type — the two were tightly coupled (the retrieve validator
// reads proxy state, the proxy needs service indexes) so they're now one struct.
type Service struct {
	cfg    ServiceConfig
	db     *gorp.DB
	closer io.MultiCloser
	Writer
	otg     *ontology.Ontology
	group   group.Group
	table   *gorp.Table[Key, Channel]
	indexes indexes
	// leasedCounter and freeCounter drive local-key assignment for
	// gateway-leased and free-virtual channels respectively. freeCounter is
	// only populated on the bootstrapper node.
	leasedCounter *counter
	freeCounter   *counter
	// mu guards externalNonVirtualSet, which tracks the key set used by
	// validateChannels to enforce the uint20 channel-index overflow limit.
	// Retrieve.Validate routes through here, and createGateway/deleteGateway
	// hold the write lock while mutating the set, so internal callers must
	// use newRetrieve (no validator) to avoid deadlocking against their own
	// pending writes.
	mu struct {
		externalNonVirtualSet *set.Integer[Key]
		sync.RWMutex
	}
	createRouter proxy.BatchFactory[Channel]
	renameRouter proxy.BatchFactory[renameBatchEntry]
	keyRouter    proxy.BatchFactory[Key]
}

type IntOverflowChecker = func(types.Uint20) error

type ServiceConfig struct {
	alamos.Instrumentation
	HostResolver     cluster.HostResolver
	ClusterDB        *gorp.DB
	TSChannel        *ts.DB
	Transport        Transport
	Ontology         *ontology.Ontology
	Group            *group.Service
	IntOverflowCheck IntOverflowChecker
	Search           *search.Index
	// ValidateNames sets whether to validate channel names during creation and
	// renaming.
	ValidateNames *bool
	// ForceMigration will force all migrations to run, regardless of whether they have
	// already been run.
	ForceMigration *bool
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "host_resolver", c.HostResolver)
	validate.NotNil(v, "cluster_db", c.ClusterDB)
	validate.NotNil(v, "ts_channel", c.TSChannel)
	validate.NotNil(v, "transport", c.Transport)
	validate.NotNil(v, "int_overflow_check", c.IntOverflowCheck)
	validate.NotNil(v, "validate_names", c.ValidateNames)
	validate.NotNil(v, "force_migration", c.ForceMigration)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.ClusterDB = override.Nil(c.ClusterDB, other.ClusterDB)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.IntOverflowCheck = override.Nil(c.IntOverflowCheck, other.IntOverflowCheck)
	c.Search = override.Nil(c.Search, other.Search)
	c.ValidateNames = override.Nil(c.ValidateNames, other.ValidateNames)
	c.ForceMigration = override.Nil(c.ForceMigration, other.ForceMigration)
	return c
}

var DefaultServiceConfig = ServiceConfig{ValidateNames: new(true), ForceMigration: new(false)}

func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:          cfg,
		db:           cfg.ClusterDB,
		otg:          cfg.Ontology,
		indexes:      newIndexes(),
		createRouter: proxy.BatchFactory[Channel]{Host: cfg.HostResolver.HostKey()},
		keyRouter:    proxy.BatchFactory[Key]{Host: cfg.HostResolver.HostKey()},
		renameRouter: proxy.BatchFactory[renameBatchEntry]{Host: cfg.HostResolver.HostKey()},
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Channel]{
		DB:              cfg.ClusterDB,
		Migrations:      []migrate.Migration{gorp.CodecMigration[Key, Channel]("msgpack_to_orc")},
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if cfg.Group != nil {
		if s.group, err = cfg.Group.CreateOrRetrieve(ctx, "Channels", ontology.RootID); !ok(err, nil) {
			return nil, err
		}
	}
	leasedCounterKey := []byte(cfg.HostResolver.HostKey().String() + ".distribution.channel.leasedCounter")
	if s.leasedCounter, err = openCounter(ctx, cfg.ClusterDB, leasedCounterKey); !ok(err, nil) {
		return nil, err
	}
	// Seed the external/non-virtual key set by scanning the table once at
	// startup. This is the only call site that unavoidably walks the table —
	// there is no index for (Internal, Virtual) and the cost is bounded by
	// how many channels the cluster has accumulated.
	var externalNonVirtualChannels []Channel
	if err = s.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
			return !c.Internal && !c.Virtual, nil
		})).
		Entries(&externalNonVirtualChannels).
		Exec(ctx, cfg.ClusterDB); !ok(err, nil) {
		return nil, err
	}
	s.mu.externalNonVirtualSet = set.NewInteger(KeysFromChannels(externalNonVirtualChannels))
	if cfg.HostResolver.HostKey() == cluster.NodeKeyBootstrapper {
		freeCounterKey := []byte(cfg.HostResolver.HostKey().String() + ".distribution.channel.counter.free")
		if s.freeCounter, err = openCounter(ctx, cfg.ClusterDB, freeCounterKey); !ok(err, nil) {
			return nil, err
		}
	}
	cfg.Transport.CreateServer().BindHandler(s.createHandler)
	cfg.Transport.DeleteServer().BindHandler(s.deleteHandler)
	cfg.Transport.RenameServer().BindHandler(s.renameHandler)
	s.Writer = s.NewWriter(nil)
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	cfg.Search.RegisterService(s)
	return s, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: s.db.OverrideTx(tx)}
}

func (s *Service) Group() group.Group { return s.group }

// Observe returns an observable that notifies callers of changes to channel entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Channel]] {
	return s.table.Observe()
}

// newRetrieve returns a Retrieve without the channel-index overflow
// validator attached. Internal callers (create / delete / rename paths)
// must use this instead of NewRetrieve because they run inside the write
// window that validateChannels' RLock would block on, and because they
// don't need the overflow check — they already enforce it inline at commit
// time.
func (s *Service) newRetrieve() Retrieve {
	return Retrieve{
		baseTX:  s.db,
		gorp:    s.table.NewRetrieve(),
		search:  s.cfg.Search,
		indexes: s.indexes,
	}
}

// NewRetrieve opens a retrieve query for external callers, with the channel
// index overflow validator attached to enforce the uint20 cap on retrieved
// external non-virtual channels.
func (s *Service) NewRetrieve() Retrieve {
	r := s.newRetrieve()
	r.gorp = r.gorp.Validate(s.validateChannels)
	return r
}

// CountExternalNonVirtual returns the number of external non-virtual channels in the
// service.
func (s *Service) CountExternalNonVirtual() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return uint32(s.mu.externalNonVirtualSet.Size())
}

func (s *Service) Close() error { return s.closer.Close() }

// validateChannels runs after every Retrieve.Exec (when called via
// NewRetrieve) and fails the query if any retrieved external non-virtual
// channel would push the uint20 channel index past the configured overflow
// limit. Attached via Validate on NewRetrieve so it propagates through the
// generated Exec/Count/Exists pipeline.
func (s *Service) validateChannels(_ gorp.Context, channels []Channel) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range channels {
		key := ch.GorpKey()
		if !s.mu.externalNonVirtualSet.Contains(key) {
			continue
		}
		channelNumber := s.mu.externalNonVirtualSet.NumLessThan(key) + 1
		if err := s.cfg.IntOverflowCheck(types.Uint20(channelNumber)); err != nil {
			return err
		}
	}
	return nil
}

func TryToRetrieveStringer(ctx context.Context, svc *Service, key Key) string {
	if svc == nil {
		return key.String()
	}
	var ch Channel
	if err := svc.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
		return key.String()
	}
	return ch.String()
}
