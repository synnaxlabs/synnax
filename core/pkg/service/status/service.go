// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"slices"
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status/versions"
	v2 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	// DB is the underlying database that the service will use to store Statuses.
	//
	// [REQUIRED]
	DB *gorp.DB
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = (*ServiceConfig)(nil)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config
func (c ServiceConfig) Validate() error {
	v := validate.New("service.status")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service is the main entrypoint for managing statuses within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting statuses. It also
// provides mechanisms for listening to changes in statuses.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Status[any]]
	indexes indexes
	state   *actions.State[Key, Action]
}

// OpenService opens a new Service with the provided configuration. If error is nil, the
// service is ready for use and must be closed by calling Close to prevent resource
// leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		indexes: newIndexes(),
		state:   actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(
		ctx,
		gorp.TableConfig[Key, Status[any]]{
			DB:              cfg.DB,
			Name:            tableName,
			Indexes:         s.indexes.all(),
			References:      s.indexes.references(),
			Instrumentation: cfg.Instrumentation,
			Migrations:      versions.Migrations,
			// A label delete detaches the label from each status that holds it, which
			// no status write does, so the service sets the status that remains. The
			// device, rack, and channel tables need no hook, as a group delete refuses
			// a group that holds members.
			OnDetach: func(_ context.Context, tx gorp.Tx, st Status[any]) error {
				s.state.Dispatcher().
					Attach(tx, st.Key, NewSetAction(SetPayload{Status: st}))
				return nil
			},
		},
	); !ok(err, s.table) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases any resources that it may have acquired. Close
// is not safe to call concurrently with any other Service methods (including Writer(s)
// and Retrieve(s)).
func (s *Service) Close() error { return s.closer.Close() }

// Observe returns an observable that notifies callers of changes to status entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Status[any]]] {
	return s.table.Observe()
}

// Changes returns the changes that c made to statuses.
func (s *Service) Changes(
	ctx context.Context,
	c gorp.Commit,
) gorp.TxReader[Key, Status[any]] {
	return s.table.Changes(ctx, c)
}

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// ResolveKeyOrName returns all statuses matching keyOrName, preferring an exact key
// match over name matches, which come in key order. Read-only; callers enforce access
// on and write the result.
func (s *Service) ResolveKeyOrName(
	ctx context.Context,
	snap gorp.Snapshot,
	keyOrName string,
) ([]Status[any], error) {
	if keyOrName == "" {
		return nil, errors.Wrap(validate.ErrValidation, "key_or_name is required")
	}
	var st Status[any]
	err := s.NewRetrieve[any]().Where(MatchKeys[any](keyOrName)).
		Entry(&st).
		Exec(ctx, snap)
	if err == nil {
		return []Status[any]{st}, nil
	}
	if !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	var matches []Status[any]
	if err = s.NewRetrieve[any]().
		Where(MatchNames[any](keyOrName)).
		Entries(&matches).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	slices.SortFunc(matches, func(a, b Status[any]) int {
		return strings.Compare(a.Key, b.Key)
	})
	return matches, nil
}

// SetTarget builds the status a by-key-or-name set should write: the first match, or a
// new UUID-keyed status named keyOrName when nothing matched, with the new fields
// applied. It clears the description, which explained the previous message.
func SetTarget(matches []Status[any], keyOrName, message, variant string) Status[any] {
	st := Status[any]{Key: uuid.New().String(), Name: keyOrName}
	if len(matches) > 0 {
		st = matches[0]
	}
	st.Message = message
	st.Description = ""
	st.Variant = Variant(variant)
	st.Time = telem.Now()
	return st
}

// SetByKeyOrName updates the first status matching keyOrName by key or name, creating a
// new status named keyOrName when none match. It returns the written key, reports
// multipleMatches when more than one name matched, and returns a validate.ErrValidation
// for an unknown variant.
func (s *Service) SetByKeyOrName(
	ctx context.Context,
	keyOrName, message, variant string,
) (key Key, multipleMatches bool, err error) {
	// Check before opening a Tx
	if !Variant(variant).IsValid() {
		return "", false, errors.Wrap(validate.ErrValidation, "invalid status variant")
	}
	if err = s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		matches, err := s.ResolveKeyOrName(ctx, tx, keyOrName)
		if err != nil {
			return err
		}
		st := SetTarget(matches, keyOrName, message, variant)
		if err = s.NewWriter(tx).Set(ctx, &st); err != nil {
			return err
		}
		key, multipleMatches = st.Key, len(matches) > 1
		return nil
	}); err != nil {
		return "", false, err
	}
	return key, multipleMatches, nil
}

// SetMissing writes each of statuses whose owner holds no status, under the key that
// v2.MigrateKey derives from the owner. Each status needs an owner. It skips reference
// checks, so only the migrations of the tables that status owners live in may call it.
func (s *Service) SetMissing[D any](
	ctx context.Context,
	tx gorp.Tx,
	statuses []Status[D],
) error {
	owners := make([]resource.ID, len(statuses))
	for i, st := range statuses {
		owners[i] = *st.Owner
	}
	var held []Status[D]
	if err := s.NewRetrieve[D]().
		Where(MatchOwners[D](owners...)).
		Entries(&held).
		Exec(ctx, tx); err != nil {
		return err
	}
	w := s.NewMigrationWriter(tx)
	for _, st := range statuses {
		if slices.ContainsFunc(held, func(h Status[D]) bool {
			return *h.Owner == *st.Owner
		}) {
			continue
		}
		st.Key, _ = v2.MigrateKey(st.Owner.String())
		if err := w.Set(ctx, st.ToAny()); err != nil {
			return err
		}
	}
	return nil
}

// NewMigrationWriter returns a writer that skips reference checks, for the migrations
// of the tables that status owners live in. Those migrations run before their tables
// open, so no other caller may use it.
func (s *Service) NewMigrationWriter(tx gorp.Tx) *gorp.Writer[Key, Status[any]] {
	return s.table.NewUncheckedWriter(tx)
}

// NewWriter opens a Writer for statuses that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		indexes:    s.indexes,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a Retrieve query for statuses whose details are of type D.
func (s *Service) NewRetrieve[D any]() Retrieve[D] {
	return Retrieve[D]{
		gorp:    gorp.NewRetrieve[Key, Status[D]](),
		indexes: s.indexes,
	}
}

// Join returns the status that each resource in ids owns, in the same order. A resource
// that owns no status gets nil.
func (s *Service) Join[D any](
	ctx context.Context,
	snap gorp.Snapshot,
	ids []resource.ID,
) ([]*Status[D], error) {
	var statuses []Status[D]
	if err := s.NewRetrieve[D]().
		Where(MatchOwners[D](ids...)).
		Entries(&statuses).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	byOwner := make(map[resource.ID]*Status[D], len(statuses))
	for i := range statuses {
		if owner := statuses[i].Owner; owner != nil {
			byOwner[*owner] = &statuses[i]
		}
	}
	joined := make([]*Status[D], len(ids))
	for i, id := range ids {
		joined[i] = byOwner[id]
	}
	return joined, nil
}
