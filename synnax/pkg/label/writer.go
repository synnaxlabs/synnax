/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package label

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
}

func (w Writer) Create(
	ctx context.Context,
	l *Label,
) (err error) {
	if l.Key == uuid.Nil {
		l.Key = uuid.New()
	}
	if err = gorp.NewCreate[uuid.UUID, Label]().Entry(l).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(l.Key)
	return w.otg.DefineResource(ctx, otgID)
}

func (w Writer) CreateMany(
	ctx context.Context,
	ls *[]Label,
) (err error) {
	for i, l := range *ls {
		if err = w.Create(ctx, &l); err != nil {
			return
		}
		(*ls)[i] = l
	}
	return err
}

func (w Writer) Update(
	ctx context.Context,
	l Label,
) (err error) {
	return gorp.NewUpdate[uuid.UUID, Label]().WhereKeys(l.Key).Change(func(ol Label) Label {
		if l.Name != "" {
			ol.Name = l.Name
		}
		if !l.Color.IsZero() {
			ol.Color = l.Color
		}
		return l
	}).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	k uuid.UUID,
) (err error) {
	if err = gorp.NewDelete[uuid.UUID, Label]().WhereKeys(k).Exec(ctx, w.tx); err != nil {
		return
	}
	return w.otg.DeleteResource(ctx, OntologyID(k))
}

func (w Writer) DeleteMany(
	ctx context.Context,
	ks []uuid.UUID,
) (err error) {
	for _, k := range ks {
		if err = w.Delete(ctx, k); err != nil {
			return
		}
	}
	return err
}

func (w Writer) Label(
	ctx context.Context,
	target ontology.ID,
	labels []uuid.UUID,
) error {
	for _, label := range labels {
		if err := w.otg.DefineRelationship(ctx, target, LabeledBy, OntologyID(label)); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) RemoveLabel(
	ctx context.Context,
	target ontology.ID,
	labels []uuid.UUID,
) error {
	for _, label := range labels {
		if err := w.otg.DeleteRelationship(ctx, target, LabeledBy, OntologyID(label)); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) validate(l Label) error {
	v := validate.New("label.Label")
	validate.NotNil(v, "Task", l.Key)
	validate.NotEmptyString(v, "Name", l.Name)
	validate.NonZeroable(v, "Color", l.Color)
	return v.Error()
}
