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
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
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
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// IntOverflowChecker reports whether a channel causes an integer overflow.
type IntOverflowChecker = func(types.Uint20) error

// ServiceConfig configures the service-layer channel service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Channel is the distribution-layer channel service the service drives to assign
	// local keys and create, rename, and delete storage channels across the cluster.
	//
	// [REQUIRED]
	Channel *channel.Service
	// DB is the cluster-wide metadata database backing the channel table.
	//
	// [REQUIRED]
	DB *gorp.DB
	// HostResolver provides this node's key for default leaseholder assignment.
	//
	// [REQUIRED]
	HostResolver node.HostResolver
	// Ontology integrates channels into the resource ontology.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is the channel group resources are parented under.
	//
	// [REQUIRED]
	Group *group.Service
	// Search is the full-text search index channels are registered with.
	//
	// [REQUIRED]
	Search *search.Index
	// IntOverflowCheck enforces the cap on external (non-internal, non-virtual)
	// channels.
	//
	// [OPTIONAL] - Defaults to system integer overflow check.
	IntOverflowCheck IntOverflowChecker
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
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "host_resolver", c.HostResolver)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "search", c.Search)
	validate.NotNil(v, "int_overflow_check", c.IntOverflowCheck)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "validate_names", c.ValidateNames)
	return v.Error()
}

// Override implements config.Config.
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

// Service is the top-level channel service. It owns the channel metadata table,
// retrieval, ontology and search integration, and the create/delete/rename
// orchestration, driving the distribution-layer allocator for key assignment and
// storage. It also infers DataTypes for calculated channels on write.
type Service struct {
	cfg     ServiceConfig
	db      *gorp.DB
	closer  xio.MultiCloser
	group   group.Group
	table   *gorp.Table[Key, Channel]
	indexes indexes
	// mu guards the external non-virtual channel key set, which is shared across every
	// Writer the service spawns and the retrieve-time validator.
	mu struct {
		sync.RWMutex
		// externalNonVirtualSet tracks the keys of external (non-internal, non-virtual)
		// channels. The create path and the retrieve-time overflow validator consult it
		// to enforce the uint20 channel-index overflow limit.
		externalNonVirtualSet set.Integer[Key]
	}
}

// OpenService opens a channel service using the provided configuration(s).
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{
		ValidateNames:    new(true),
		IntOverflowCheck: verification.DefaultOverflowCheck,
	}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg, db: cfg.DB, indexes: newIndexes()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Channel]{
		DB: cfg.DB,
		Migrations: []migrate.Migration{
			gorp.CodecMigration[Key, Channel]("msgpack_to_orc"),
		},
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(
		ctx, "Channels", ontology.RootID,
	); !ok(err, nil) {
		return nil, err
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
	s.mu.externalNonVirtualSet.Insert(KeysFromChannels(externalNonVirtualChannels)...)
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	return s, nil
}

// Group returns the group under which the service's channels are created.
func (s *Service) Group() group.Group { return s.group }

// newRetrieve returns a Retrieve without the channel-index overflow validator attached.
// Internal callers (the create / delete / rename paths) use this instead of NewRetrieve
// because they run inside the write window that validateChannels' RLock would block on,
// and because they enforce the overflow check inline at commit time.
func (s *Service) newRetrieve() Retrieve {
	return Retrieve{
		baseTX:  s.db,
		gorp:    s.table.NewRetrieve(),
		search:  s.cfg.Search,
		indexes: s.indexes,
	}
}

// Observe returns an observable that notifies callers of changes to channels.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Channel]] {
	return s.table.Observe()
}

// NewRetrieve opens a retrieve query for external callers, with the channel index
// overflow validator attached to enforce the uint20 cap on retrieved external
// non-virtual channels.
func (s *Service) NewRetrieve() Retrieve {
	r := s.newRetrieve()
	r.gorp = r.gorp.Validate(s.validateChannels)
	return r
}

// Close releases the resources held by the Service.
func (s *Service) Close() error { return s.closer.Close() }

// validateChannels runs after every Retrieve.Exec (when called via NewRetrieve) and
// fails the query if any retrieved external non-virtual channel would push the uint20
// channel index past the configured overflow limit.
func (s *Service) validateChannels(_ gorp.Context, channels []Channel) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range channels {
		key := ch.Key()
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

// NewWriter returns a Writer scoped to the provided transaction (nil writes directly to
// the service DB).
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = s.db.OverrideTx(tx)
	return Writer{
		svc: s,
		tx:  tx,
		otg: s.cfg.Ontology.NewWriter(tx),
		analyzer: NewCalculationAnalyzer(s.NewArcSymbolResolver(tx), parser.Config{
			AllowDashedNames: !s.ShouldValidateNames(),
		}),
	}
}

// ShouldValidateNames reports whether channel-name validation is enabled.
func (s *Service) ShouldValidateNames() bool { return *s.cfg.ValidateNames }

// The methods below are convenience shortcuts that run a single operation through a
// fresh, transaction-less Writer. Callers that need to batch writes into a transaction
// should use NewWriter directly.

// Create creates a single channel outside of any transaction, validating a calculated
// channel's expression up front (an analysis error aborts the create). It is a
// convenience wrapper over a fresh, transaction-less Writer; see Writer.Create.
func (s *Service) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	channels := []Channel{*c}
	if err := s.CreateMany(ctx, &channels, opts...); err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

// CreateMany creates multiple channels outside of any transaction, validating calculated
// channel expressions up front (an analysis error aborts the call). It is a convenience
// wrapper over a fresh, transaction-less Writer; see Writer.CreateMany.
func (s *Service) CreateMany(
	ctx context.Context, channels *[]Channel, opts ...CreateOption,
) error {
	return s.NewWriter(nil).CreateMany(ctx, channels, opts...)
}
