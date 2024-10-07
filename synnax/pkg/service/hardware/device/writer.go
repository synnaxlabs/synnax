/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package device

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
}

func (w Writer) Create(ctx context.Context, d Device) (err error) {
	if err := d.Validate(); err != nil {
		return err
	}
	if err = gorp.NewCreate[string, Device]().Entry(&d).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(d.Key)
	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return
	}
	return w.otg.DefineRelationship(ctx, w.group.OntologyID(), ontology.ParentOf, otgID)
}

func (w Writer) Delete(ctx context.Context, key string) error {
	if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
		return err
	}
	return gorp.NewDelete[string, Device]().WhereKeys(key).Exec(ctx, w.tx)
}
