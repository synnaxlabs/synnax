// Copyright 2026 Synnax Labs, Inc.
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
	"fmt"
	"io"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// ServiceConfig is the configuration for opening a Service. D is the resource type
// the service stores.
type ServiceConfig[D gorp.Entry[uuid.UUID]] struct {
	// DB is the database resources are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology registers the resource type and holds each resource's parent edge.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Status stores each resource's status.
	// [REQUIRED]
	Status *status.Service
	// Signals publishes set and delete events on sy_<type>_set and
	// sy_<type>_delete.
	// [OPTIONAL]
	Signals *signals.Provider
	// Type is the ontology resource type of the stored resources.
	// [REQUIRED]
	Type ontology.ResourceType
	// Schema dumps a resource into its ontology resource data.
	// [REQUIRED]
	Schema zyn.Schema
	// Indexes are the secondary indexes on the resource table, Search among them.
	// [REQUIRED]
	Indexes []gorp.Index[uuid.UUID, D]
	// Search is the index that ontology searches match resources through.
	// [REQUIRED]
	Search *gorp.SearchIndex[uuid.UUID, D]
	// Migrations is the stored-shape migration chain for the resource type. Empty
	// for a type whose stored shape has never changed.
	Migrations []gorp.Migration
	// SetEntryKey stamps key onto a resource.
	// [REQUIRED]
	SetEntryKey func(d *D, key uuid.UUID)
	// NameOf returns a resource's name.
	// [REQUIRED]
	NameOf func(D) string
	// RackOf returns the rack a resource belongs to.
	// [REQUIRED]
	RackOf func(D) rack.Key
	// ParentOf returns the ontology ID of a resource's parent, and false when the
	// parent is the resource's rack.
	// [OPTIONAL] - nil when every resource is a child of its rack.
	ParentOf func(D) (ontology.ID, bool)
	// StatusDetails returns the details carried by a resource's initial status.
	// [OPTIONAL] - nil when the initial status carries no details.
	StatusDetails func(D) any
	// ValidateEntry checks a resource before it is stored.
	// [OPTIONAL] - nil when the resource type has no validation rules.
	ValidateEntry func(*D) error
	// Referenced reports whether a task still references the resource under key.
	// Delete refuses while it returns true.
	// [OPTIONAL] - nil when no task references the resource type.
	Referenced func(ctx context.Context, tx gorp.Tx, key uuid.UUID) (bool, error)
	// DeleteChildren deletes the resources whose parent is the resource under key.
	// [OPTIONAL] - nil when the resource type has no children.
	DeleteChildren func(ctx context.Context, tx gorp.Tx, key uuid.UUID) error
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig[entry]] = ServiceConfig[entry]{}

// Override implements config.Config.
func (c ServiceConfig[D]) Override(other ServiceConfig[D]) ServiceConfig[D] {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Type = override.String(c.Type, other.Type)
	c.Schema = override.Nil(c.Schema, other.Schema)
	c.Indexes = override.Slice(c.Indexes, other.Indexes)
	c.Search = override.Nil(c.Search, other.Search)
	c.Migrations = override.Slice(c.Migrations, other.Migrations)
	c.SetEntryKey = override.Nil(c.SetEntryKey, other.SetEntryKey)
	c.NameOf = override.Nil(c.NameOf, other.NameOf)
	c.RackOf = override.Nil(c.RackOf, other.RackOf)
	c.ParentOf = override.Nil(c.ParentOf, other.ParentOf)
	c.StatusDetails = override.Nil(c.StatusDetails, other.StatusDetails)
	c.ValidateEntry = override.Nil(c.ValidateEntry, other.ValidateEntry)
	c.Referenced = override.Nil(c.Referenced, other.Referenced)
	c.DeleteChildren = override.Nil(c.DeleteChildren, other.DeleteChildren)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig[D]) Validate() error {
	v := validate.New("device")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("status", c.Status)
	v.NotEmptyString("type", c.Type)
	v.NotNil("schema", c.Schema)
	v.NotEmptySlice("indexes", c.Indexes)
	v.NotNil("search", c.Search)
	v.NotNil("set_entry_key", c.SetEntryKey)
	v.NotNil("name_of", c.NameOf)
	v.NotNil("rack_of", c.RackOf)
	return v.Error()
}

// Service stores the resources of one hardware resource type: a chassis, a module,
// a server, and so on. D is the resource type, e.g. Service[ni.Device].
type Service[D gorp.Entry[uuid.UUID]] struct {
	cfg    ServiceConfig[D]
	closer xio.MultiCloser
	table  *gorp.Table[uuid.UUID, D]
}

// OpenService opens a Service for one resource type. The service must be closed by
// calling Close.
func OpenService[D gorp.Entry[uuid.UUID]](
	ctx context.Context,
	cfgs ...ServiceConfig[D],
) (s *Service[D], err error) {
	cfg, err := config.New(ServiceConfig[D]{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service[D]{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, D]{
		DB:              cfg.DB,
		Indexes:         cfg.Indexes,
		Migrations:      cfg.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		pub := signals.GorpPublisherConfigUUID(s.table.Observe())
		pub.SetName = fmt.Sprintf("sy_%s_set", cfg.Type)
		pub.DeleteName = fmt.Sprintf("sy_%s_delete", cfg.Type)
		var sig io.Closer
		if sig, err = cfg.Signals.PublishFromGorp(ctx, pub); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service[D]) Close() error { return s.closer.Close() }

// OntologyID returns the ontology ID of the resource under key.
func (s *Service[D]) OntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: s.cfg.Type, Key: key.String()}
}

// NewRetrieve opens a query over the stored resources.
func (s *Service[D]) NewRetrieve() gorp.Retrieve[uuid.UUID, D] {
	return s.table.NewRetrieve()
}

// Create stores d, defines its ontology resource and parent edge, and writes an
// unknown status when it has none. A resource with a nil key gets a new one. It
// returns the resource's validation error when it fails validation. A nil tx runs the
// writes in their own transaction.
func (s *Service[D]) Create(ctx context.Context, tx gorp.Tx, d *D) error {
	if tx == nil {
		return s.cfg.DB.WithTx(
			ctx,
			func(tx gorp.Tx) error { return s.Create(ctx, tx, d) },
		)
	}
	if (*d).GorpKey() == uuid.Nil() {
		s.cfg.SetEntryKey(d, uuid.New())
	}
	if s.cfg.ValidateEntry != nil {
		if err := s.cfg.ValidateEntry(d); err != nil {
			return err
		}
	}
	if err := s.table.NewCreate().Entry(d).Exec(ctx, tx); err != nil {
		return err
	}
	id := s.OntologyID((*d).GorpKey())
	otg := s.cfg.Ontology.NewWriter(tx)
	if err := otg.DefineResources(ctx, id); err != nil {
		return err
	}
	if err := s.setParent(ctx, otg, id, *d); err != nil {
		return err
	}
	exists, err := gorp.NewRetrieve[status.Key, status.Status[any]]().
		Where(gorp.MatchKeys[status.Key, status.Status[any]](id.String())).
		Exists(ctx, tx)
	if err != nil || exists {
		return err
	}
	unknown := s.unknownStatus(*d)
	return s.cfg.Status.NewWriter(tx).Set(ctx, &unknown)
}

// Update applies change to the resource stored under key and stores the result.
// The resource's status and parent edge follow its name and parent. It returns an
// error wrapping query.ErrNotFound when no resource is stored under key, and the
// resource's validation error when the result fails validation. A nil tx runs the
// writes in their own transaction.
func (s *Service[D]) Update(
	ctx context.Context,
	tx gorp.Tx,
	key uuid.UUID,
	change func(D) (D, error),
) error {
	if tx == nil {
		return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return s.Update(ctx, tx, key, change)
		})
	}
	var prev, next D
	if err := s.table.NewUpdate().
		Where(gorp.MatchKeys[uuid.UUID, D](key)).
		ChangeErr(func(_ gorp.Context, d D) (D, error) {
			prev = d
			var zero D
			updated, err := change(d)
			if err != nil {
				return zero, err
			}
			s.cfg.SetEntryKey(&updated, key)
			if s.cfg.ValidateEntry != nil {
				if err := s.cfg.ValidateEntry(&updated); err != nil {
					return zero, err
				}
			}
			next = updated
			return updated, nil
		}).
		Exec(ctx, tx); err != nil {
		return err
	}
	id := s.OntologyID(key)
	if name := s.cfg.NameOf(next); name != s.cfg.NameOf(prev) {
		if err := s.cfg.Status.NewWriter(tx).
			Rename(ctx, id.String(), name); err != nil {
			return err
		}
	}
	if s.parentOf(next) != s.parentOf(prev) {
		return s.setParent(ctx, s.cfg.Ontology.NewWriter(tx), id, next)
	}
	return nil
}

// Delete removes the resources stored under keys with their children, statuses, and
// ontology resources. It returns an error wrapping validate.ErrValidation when a
// task still references one of them. A nil tx runs the writes in their own
// transaction. Delete is idempotent.
func (s *Service[D]) Delete(ctx context.Context, tx gorp.Tx, keys ...uuid.UUID) error {
	if tx == nil {
		return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return s.Delete(ctx, tx, keys...)
		})
	}
	if s.cfg.Referenced != nil {
		for _, key := range keys {
			referenced, err := s.cfg.Referenced(ctx, tx, key)
			if err != nil {
				return err
			}
			if referenced {
				return errors.Wrapf(
					validate.ErrValidation,
					"%s %s is referenced by a task",
					s.cfg.Type,
					key,
				)
			}
		}
	}
	otg := s.cfg.Ontology.NewWriter(tx)
	stat := s.cfg.Status.NewWriter(tx)
	for _, key := range keys {
		if s.cfg.DeleteChildren != nil {
			if err := s.cfg.DeleteChildren(ctx, tx, key); err != nil {
				return err
			}
		}
		id := s.OntologyID(key)
		if err := stat.Delete(ctx, id.String()); err != nil {
			return err
		}
		if err := otg.DeleteResources(ctx, id); err != nil {
			return err
		}
	}
	return errors.Skip(
		s.table.NewDelete().Where(gorp.MatchKeys[uuid.UUID, D](keys...)).Exec(ctx, tx),
		query.ErrNotFound,
	)
}

func (s *Service[D]) parentOf(d D) ontology.ID {
	if s.cfg.ParentOf != nil {
		if parent, ok := s.cfg.ParentOf(d); ok {
			return parent
		}
	}
	return s.cfg.RackOf(d).OntologyID()
}

func (s *Service[D]) setParent(
	ctx context.Context,
	otg ontology.Writer,
	id ontology.ID,
	d D,
) error {
	if err := otg.DeleteIncomingRelationshipsOfType(
		ctx,
		id,
		ontology.RelationshipTypeParentOf,
	); err != nil {
		return err
	}
	return otg.DefineRelationships(
		ctx,
		s.parentOf(d),
		ontology.RelationshipTypeParentOf,
		id,
	)
}

func (s *Service[D]) unknownStatus(d D) status.Status[any] {
	name := s.cfg.NameOf(d)
	var details any
	if s.cfg.StatusDetails != nil {
		details = s.cfg.StatusDetails(d)
	}
	return status.Status[any]{
		Key:     s.OntologyID(d.GorpKey()).String(),
		Name:    name,
		Time:    telem.Now(),
		Variant: status.VariantWarning,
		Message: "Status unknown",
		Details: details,
	}
}
