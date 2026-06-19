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
	"github.com/synnaxlabs/arc"
	dischannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	xservice "github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// IntOverflowChecker reports an error when the provided channel index would exceed the
// permitted number of external channels.
type IntOverflowChecker = func(types.Uint20) error

// ServiceConfig configures the service-layer channel service.
type ServiceConfig struct {
	alamos.Instrumentation
	// Channel is the distribution-layer channel service the service drives to assign
	// local keys and create, rename, and delete storage channels across the cluster.
	Channel *dischannel.Service
	// DB is the cluster-wide metadata database backing the channel table.
	DB *gorp.DB
	// HostResolver provides this node's key for default leaseholder assignment.
	HostResolver node.HostResolver
	// Ontology integrates channels into the resource ontology.
	Ontology *ontology.Ontology
	// Group is the channel group resources are parented under.
	Group *group.Service
	// Search is the full-text search index channels are registered with.
	Search *search.Index
	// IntOverflowCheck enforces the cap on external (non-internal, non-virtual) channels.
	IntOverflowCheck IntOverflowChecker
	// Status publishes error/clear statuses for calculated channels.
	Status *status.Service
	// ValidateNames sets whether to validate channel names during creation and renaming.
	ValidateNames *bool
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("service.channel")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "host_resolver", c.HostResolver)
	validate.NotNil(v, "int_overflow_check", c.IntOverflowCheck)
	validate.NotNil(v, "validate_names", c.ValidateNames)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.DB = override.Nil(c.DB, other.DB)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Search = override.Nil(c.Search, other.Search)
	c.IntOverflowCheck = override.Nil(c.IntOverflowCheck, other.IntOverflowCheck)
	c.Status = override.Nil(c.Status, other.Status)
	c.ValidateNames = override.Nil(c.ValidateNames, other.ValidateNames)
	return c
}

// DefaultServiceConfig is the default configuration for the channel service. The overflow
// check defaults to the free tier; production overrides it with the verification service.
var DefaultServiceConfig = ServiceConfig{
	ValidateNames:    new(true),
	IntOverflowCheck: verification.FreeOverflowCheck,
}

// Service is the top-level channel service. It owns the channel metadata table,
// retrieval, ontology and search integration, and the create/delete/rename
// orchestration, driving the distribution-layer allocator for key assignment and
// storage. It also infers DataTypes for calculated channels on write.
type Service struct {
	cfg    ServiceConfig
	db     *gorp.DB
	closer xio.MultiCloser
	Writer
	otg     *ontology.Ontology
	group   group.Group
	table   *gorp.Table[Key, Channel]
	indexes indexes
	// mu guards externalNonVirtualSet, which tracks the key set used by
	// validateChannels to enforce the uint20 channel-index overflow limit.
	mu struct {
		externalNonVirtualSet *set.Integer[Key]
		sync.RWMutex
	}
}

// OpenService opens a channel service using the provided configuration(s).
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		db:      cfg.DB,
		otg:     cfg.Ontology,
		indexes: newIndexes(),
	}
	cleanup, ok := xservice.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Channel]{
		DB:              cfg.DB,
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
	// Seed the external/non-virtual key set by scanning the table once at startup. This
	// is the only call site that unavoidably walks the table — there is no index for
	// (Internal, Virtual) and the cost is bounded by how many channels the cluster has
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
	s.mu.externalNonVirtualSet = set.NewInteger(KeysFromChannels(externalNonVirtualChannels))
	s.Writer = s.NewWriter(nil)
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	cfg.Search.RegisterService(s)
	return s, nil
}

func (s *Service) Group() group.Group { return s.group }

// Observe returns an observable that notifies callers of changes to channel entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Channel]] {
	return s.table.Observe()
}

// newRetrieve returns a Retrieve without the channel-index overflow validator attached.
// Internal callers (create / delete / rename paths) must use this instead of
// NewRetrieve because they run inside the write window that validateChannels' RLock
// would block on, and because they don't need the overflow check — they already enforce
// it inline at commit time.
func (s *Service) newRetrieve() Retrieve {
	return Retrieve{
		baseTX:  s.db,
		gorp:    s.table.NewRetrieve(),
		search:  s.cfg.Search,
		indexes: s.indexes,
	}
}

// NewRetrieve opens a retrieve query for external callers, with the channel index
// overflow validator attached to enforce the uint20 cap on retrieved external
// non-virtual channels.
func (s *Service) NewRetrieve() Retrieve {
	r := s.newRetrieve()
	r.gorp = r.gorp.Validate(s.validateChannels)
	return r
}

// RetrieveDataTypes resolves the data types of the channels with the given keys,
// returning them in the same order as keys. Keys that do not correspond to an existing
// channel are omitted, so the returned slice is shorter than keys when any key is
// unknown. Its signature satisfies codec.Resolver, allowing a dynamic framer codec to
// resolve channel data types directly through the service layer.
func (s *Service) RetrieveDataTypes(
	ctx context.Context,
	keys Keys,
) ([]telem.DataType, error) {
	var channels []Channel
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	dataTypeByKey := make(map[Key]telem.DataType, len(channels))
	for _, ch := range channels {
		dataTypeByKey[ch.Key()] = ch.DataType
	}
	dataTypes := make([]telem.DataType, 0, len(keys))
	for _, key := range keys {
		if dt, ok := dataTypeByKey[key]; ok {
			dataTypes = append(dataTypes, dt)
		}
	}
	return dataTypes, nil
}

// RetrieveName resolves the name of the channel with the given key, returning an empty
// string if no channel with the key exists. Its signature satisfies codec.Resolver.
func (s *Service) RetrieveName(ctx context.Context, key Key) string {
	var ch Channel
	if err := s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, nil); err != nil {
		return ""
	}
	return ch.Name
}

// CountExternalNonVirtual returns the number of external non-virtual channels in the
// service.
func (s *Service) CountExternalNonVirtual() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return uint32(s.mu.externalNonVirtualSet.Size())
}

func (s *Service) Close() error { return s.closer.Close() }

// validateChannels runs after every Retrieve.Exec (when called via NewRetrieve) and
// fails the query if any retrieved external non-virtual channel would push the uint20
// channel index past the configured overflow limit.
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

// NewArcSymbolResolver returns a resolver that maps cluster channels to Arc symbols by
// name or numeric key, for analyzing and compiling Arc expressions such as calculated
// channels. tx scopes channel lookups; nil consults the service DB directly.
func (s *Service) NewArcSymbolResolver(tx gorp.Tx) arc.SymbolResolver {
	return &symbolResolver{svc: s, tx: tx}
}
