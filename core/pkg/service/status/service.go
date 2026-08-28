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

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status/versions"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
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
	// Ontology will be used to create relationships between statuses (parent-child) and
	// with other resources within the Synnax cluster.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create the top level "Statuses" group that will be the default
	// parent of all statuses.
	//
	// [REQUIRED]
	Group *group.Service
	// Label is used to create and manage labels for statuses.
	//
	// [REQUIRED]
	Label *label.Service
	// Search is the search index for fuzzy searching statuses.
	//
	// [REQUIRED]
	Search *search.Index
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = (*ServiceConfig)(nil)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Label = override.Nil(c.Label, other.Label)
	c.Search = override.Nil(c.Search, other.Search)
	return c
}

// Validate implements config.Config
func (c ServiceConfig) Validate() error {
	v := validate.New("service.status")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("group", c.Group)
	v.NotNil("label", c.Label)
	v.NotNil("search", c.Search)
	return v.Error()
}

// Service is the main entrypoint for managing statuses within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting statuses. It also
// provides mechanisms for listening to changes in statuses.
type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[Key, Status[any]]
	group  group.Group
}

// OpenService opens a new Service with the provided configuration. If error is nil, the
// service is ready for use and must be closed by calling Close to prevent resource
// leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(
		ctx,
		gorp.TableConfig[Key, Status[any]]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Migrations:      versions.Migrations,
		},
	); !ok(err, s.table) {
		return nil, err
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(
		ctx,
		"Statuses",
		ontology.RootID,
	); !ok(err, nil) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
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

// ResolveKeyOrName returns all statuses matching keyOrName, preferring an exact key
// match over name matches. Read-only; callers enforce access on and write the result.
func (s *Service) ResolveKeyOrName(
	ctx context.Context,
	tx gorp.Tx,
	keyOrName string,
) ([]Status[any], error) {
	if keyOrName == "" {
		return nil, errors.Wrap(validate.ErrValidation, "key_or_name is required")
	}
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	var st Status[any]
	err := s.NewRetrieve[any]().Where(MatchKeys[any](keyOrName)).
		Entry(&st).
		Exec(ctx, tx)
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
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return matches, nil
}

// SetTarget builds the status a by-key-or-name set should write: the first match, or a
// new UUID-keyed status named keyOrName when nothing matched, with the new fields
// applied.
func SetTarget(matches []Status[any], keyOrName, message, variant string) Status[any] {
	st := Status[any]{Key: uuid.NewString(), Name: keyOrName}
	if len(matches) > 0 {
		st = matches[0]
	}
	st.Message = message
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

// NewWriter opens a Writer for statuses whose details are of type D. Pass a nil tx to
// write directly against the service's DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otg:       s.cfg.Ontology,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		group:     s.group,
	}
}

// NewRetrieve opens a Retrieve query for statuses whose details are of type D.
func (s *Service) NewRetrieve[D any]() Retrieve[D] {
	return Retrieve[D]{
		gorp:   gorp.NewRetrieve[Key, Status[D]](),
		baseTX: s.cfg.DB,
		search: s.cfg.Search,
		label:  s.cfg.Label,
	}
}
