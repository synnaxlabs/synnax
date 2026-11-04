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
	"bytes"
	"context"
	"encoding/json"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

// storedDetails carries the details of a stored status unchanged in either stored
// encoding. Decoding JSON details as any would round numbers past float64 precision.
type storedDetails struct{ value any }

func (d storedDetails) MarshalJSON() ([]byte, error) { return json.Marshal(d.value) }

func (d *storedDetails) UnmarshalJSON(b []byte) error {
	d.value = json.RawMessage(bytes.Clone(b))
	return nil
}

func (d storedDetails) EncodeMsgpack(e *msgpack.Encoder) error {
	return e.Encode(d.value)
}

func (d *storedDetails) DecodeMsgpack(dec *msgpack.Decoder) (err error) {
	d.value, err = dec.DecodeInterface()
	return err
}

// Writer is used to create and update statuses within the DB. Every write touches both
// the status entry and its ontology resource, and the two land together only when the
// Writer holds a transaction spanning them.
type Writer struct {
	tx        gorp.Tx
	table     *gorp.Table[Key, Status[any]]
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
}

// Set creates or updates a status within the DB. If the Status already has a key and
// an existing Status already exists with that key, the existing status will be updated.
func (w Writer) Set[D any](ctx context.Context, s *Status[D]) error {
	return w.SetWithParent(ctx, s, ontology.ID{})
}

// SetWithParent creates or updates a status as a child of the ontology.Resource with
// the given ID. If the status already exists and a parent is provided, the existing
// parent relationship will be deleted and a new parent relationship will be created. If
// the status already exists and no parent is provided, the existing parent relationship
// will be preserved. If an empty parent is provided, the status will be created under
// the top level "Statuses" group.
func (w Writer) SetWithParent[D any](
	ctx context.Context,
	s *Status[D],
	parent ontology.ID,
) error {
	hasParent := !parent.IsZero()
	if !hasParent {
		parent = w.group.OntologyID()
	}
	if err := w.validate(*s); err != nil {
		return err
	}
	exists, err := gorp.NewRetrieve[Key, Status[D]]().Where(gorp.MatchKeys[Key, Status[D]](s.Key)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := s.OntologyID()
	if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
		return err
	}
	// Status already exists and parent provided = delete incoming relationships and
	// define new parent Status already exists and no parent provided = do nothing
	// Status does not exist = define parent
	if exists && hasParent {
		if hasRel, err := w.otg.RelationshipExists(ctx, w.tx, ontology.Relationship{
			From: parent,
			Type: ontology.RelationshipTypeParentOf,
			To:   otgID,
		}); hasRel || err != nil {
			return err
		}
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(
			ctx,
			otgID,
			ontology.RelationshipTypeParentOf,
		); err != nil {
			return err
		}
		if err = w.otgWriter.DefineRelationships(
			ctx, parent, ontology.RelationshipTypeParentOf, otgID,
		); err != nil {
			return err
		}
	} else if !exists {
		if err = w.otgWriter.DefineRelationships(
			ctx, parent, ontology.RelationshipTypeParentOf, otgID,
		); err != nil {
			return err
		}
	}
	return nil
}

// SetMany creates or updates multiple statuses within the DB. If any of the statuses
// already exist, they will be updated.
func (w Writer) SetMany[D any](ctx context.Context, statuses *[]Status[D]) error {
	return w.SetManyWithParent(ctx, statuses, ontology.ID{})
}

// SetManyWithParent creates or updates multiple statuses within the DB as child
// statuses of the ontology.Resource with the given ID. If any of the statuses already
// exist, they will be updated. If the status already exists and a parent is provided,
// the existing parent relationship will be deleted and a new parent relationship will
// be created. If the status already exists and no parent is provided, the existing
// parent relationship will be preserved. If an empty parent is provided, the status
// will be created under the top level "Statuses" group.
func (w Writer) SetManyWithParent[D any](
	ctx context.Context,
	statuses *[]Status[D],
	parent ontology.ID,
) error {
	if statuses == nil {
		return nil
	}
	for i, s := range *statuses {
		if err := w.SetWithParent(ctx, &s, parent); err != nil {
			return err
		}
		(*statuses)[i] = s
	}
	return nil
}

// Rename renames the status with the given key, keeping every other field. It returns
// query.ErrNotFound if no status with the key exists and a validation error if name is
// empty. A status written after the read and before the commit is overwritten.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	v := validate.New("status.status")
	v.NotEmptyString("name", name)
	if err := v.Error(); err != nil {
		return err
	}
	var s Status[storedDetails]
	if err := gorp.NewRetrieve[Key, Status[storedDetails]]().
		Where(gorp.MatchKeys[Key, Status[storedDetails]](key)).
		Entry(&s).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	s.Name = name
	return w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx)
}

// Delete deletes the statuses with the given keys. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Status[any]](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}

func (w Writer) validate[D any](s Status[D]) error {
	v := validate.New("status.status")
	v.NotEmptyString("key", s.Key)
	v.Positive("time", s.Time)
	v.NotEmptyString("variant", s.Variant)
	return v.Error()
}
