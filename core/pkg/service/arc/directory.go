// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"iter"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// rootName is the name of the directory that holds every Arc.
const rootName = "Arcs"

// directories stores Arc directories and serves them to the ontology.
type directories struct {
	db     *gorp.DB
	table  *gorp.Table[directory.Key, directory.Directory]
	search *gorp.SearchIndex[directory.Key, directory.Directory]
}

func openDirectories(
	ctx context.Context,
	cfg ServiceConfig,
) (directories, error) {
	d := directories{
		db: cfg.DB,
		search: gorp.NewSearchIndex[directory.Key, directory.Directory](
			"search",
			func(d *directory.Directory) []string { return []string{d.Name} },
		),
	}
	var err error
	if d.table, err = gorp.OpenTable(
		ctx,
		gorp.TableConfig[directory.Key, directory.Directory]{
			DB:              cfg.DB,
			Indexes:         []gorp.Index[directory.Key, directory.Directory]{d.search},
			Instrumentation: cfg.Instrumentation,
		},
	); err != nil {
		return directories{}, err
	}
	return d, nil
}

var _ ontology.Service = directories{}

var directorySchema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

func newDirectoryResource(d directory.Directory) ontology.Resource {
	return ontology.NewResource(directorySchema, d.OntologyID(), d.Name, d)
}

// Type implements ontology.Service.
func (directories) Type() ontology.ResourceType {
	return ontology.ResourceTypeArcDirectory
}

// RetrieveResource implements ontology.Service.
func (d directories) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var dir directory.Directory
	if err = d.table.NewRetrieve().
		Where(gorp.MatchKeys[directory.Key, directory.Directory](k)).
		Entry(&dir).
		Exec(ctx, gorp.OverrideTx(d.db, tx)); err != nil {
		return ontology.Resource{}, err
	}
	return newDirectoryResource(dir), nil
}

func translateDirectoryChange(
	c xchange.Change[directory.Key, directory.Directory],
) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     directory.OntologyID(c.Key).String(),
		Value:   newDirectoryResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (d directories) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	return d.table.Observe().OnChange(func(
		ctx context.Context,
		reader gorp.TxReader[directory.Key, directory.Directory],
	) {
		f(ctx, xiter.Map(reader, translateDirectoryChange))
	})
}

// Search implements ontology.Service.
func (d directories) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[ontology.ID],
) (search.Result[ontology.ID], error) {
	return ontology.SearchIndex(ctx, tx, d.search, directory.OntologyID, q)
}

// openRoot returns the key of the root directory under the ontology root, creating it
// when it does not exist.
func (s *Service) openRoot(ctx context.Context) (root directory.Key, err error) {
	err = s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var children []ontology.Resource
		if err := s.cfg.Ontology.NewRetrieve().
			WhereIDs(ontology.RootID).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypeArcDirectory).
			ExcludeFieldData(true).
			Entries(&children).
			Exec(ctx, tx); err != nil {
			return err
		}
		if len(children) > 0 {
			var err error
			root, err = directory.KeyFromOntologyID(children[0].ID)
			return err
		}
		d := directory.Directory{Key: uuid.New(), Name: rootName}
		if err := s.directories.table.NewCreate().Entry(&d).Exec(ctx, tx); err != nil {
			return err
		}
		root = d.Key
		w := s.owner.NewWriter(tx)
		if err := w.DefineResources(ctx, d.OntologyID()); err != nil {
			return err
		}
		return w.DefineRelationships(
			ctx,
			ontology.RootID,
			ontology.RelationshipTypeParentOf,
			d.OntologyID(),
		)
	})
	return root, err
}

// CreateDirectory creates d in the directory parent, giving d a key when it has none.
func (w Writer) CreateDirectory(
	ctx context.Context,
	d *directory.Directory,
	parent directory.Key,
) error {
	if d.Key == uuid.Nil() {
		d.Key = uuid.New()
	}
	if err := d.Validate(); err != nil {
		return err
	}
	exists, err := w.directories.NewRetrieve().
		Where(gorp.MatchKeys[directory.Key, directory.Directory](d.Key)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return errors.Wrapf(
			validate.ErrValidation, "directory %s already exists", d.Key,
		)
	}
	parentID := directory.OntologyID(parent)
	if err = w.validateEntry(ctx, parentID, d.OntologyID(), d.Name, true); err != nil {
		return err
	}
	if err = w.directories.NewCreate().Entry(d).Exec(ctx, w.tx); err != nil {
		return err
	}
	if err = w.otgWriter.DefineResources(ctx, d.OntologyID()); err != nil {
		return err
	}
	return w.otgWriter.DefineRelationships(
		ctx, parentID, ontology.RelationshipTypeParentOf, d.OntologyID(),
	)
}

// RenameDirectory gives the directory with key the given name.
func (w Writer) RenameDirectory(
	ctx context.Context,
	key directory.Key,
	name string,
) error {
	if key == w.root {
		return errRoot("renamed")
	}
	if err := w.validateName(ctx, directory.OntologyID(key), name, true); err != nil {
		return err
	}
	return w.directories.NewUpdate().
		Where(gorp.MatchKeys[directory.Key, directory.Directory](key)).
		ChangeErr(func(
			_ gorp.Context,
			d directory.Directory,
		) (directory.Directory, error) {
			d.Name = name
			return d, d.Validate()
		}).
		Exec(ctx, w.tx)
}

// DeleteDirectory deletes the empty directories with keys.
func (w Writer) DeleteDirectory(ctx context.Context, keys ...directory.Key) error {
	var dirs []directory.Directory
	if err := w.directories.NewRetrieve().
		Where(gorp.MatchKeys[directory.Key, directory.Directory](keys...)).
		Entries(&dirs).
		Exec(ctx, w.tx); errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}
	for _, d := range dirs {
		if d.Key == w.root {
			return errRoot("deleted")
		}
		var children []ontology.Resource
		if err := w.otg.NewRetrieve().
			WhereIDs(d.OntologyID()).
			TraverseTo(ontology.ChildrenTraverser).
			ExcludeFieldData(true).
			Entries(&children).
			Exec(ctx, w.tx); err != nil {
			return err
		}
		if len(children) > 0 {
			return errors.Wrapf(
				validate.ErrValidation, "directory %q is not empty", d.Name,
			)
		}
		if err := w.otgWriter.DeleteResources(ctx, d.OntologyID()); err != nil {
			return err
		}
	}
	return w.directories.NewDelete().
		Where(gorp.MatchKeys[directory.Key, directory.Directory](keys...)).
		Exec(ctx, w.tx)
}
