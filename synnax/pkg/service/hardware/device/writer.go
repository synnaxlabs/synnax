// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
}

func (w Writer) Create(ctx context.Context, d Device) (err error) {
	if err = d.Validate(); err != nil {
		return
	}
	var existing Device
	err = gorp.NewRetrieve[string, Device]().WhereKeys(d.Key).Entry(&existing).Exec(ctx, w.tx)
	isNotFound := errors.Is(err, query.NotFound)
	if err != nil && !isNotFound {
		return
	}
	exists := !isNotFound
	if err = gorp.NewCreate[string, Device]().Entry(&d).Exec(ctx, w.tx); err != nil {
		return
	}
	// If the device already exists, don't redefine the resource and relationship in
	// the ontology, as to not mess with existing groups or relationships.
	otgID := OntologyID(d.Key)
	if !exists || d.Rack != existing.Rack {
		if err = w.otg.DefineResource(ctx, otgID); err != nil {
			return
		}
		if err = w.otg.DeleteIncomingRelationshipsOfType(
			ctx,
			otgID,
			ontology.ParentOf,
		); err != nil {
			return
		}
		err := w.otg.DefineRelationship(ctx, d.Rack.OntologyID(), ontology.ParentOf, otgID)
		return err
	}
	return nil
}

func (w Writer) Delete(ctx context.Context, key string) error {
	if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
		return err
	}
	return gorp.NewDelete[string, Device]().WhereKeys(key).Exec(ctx, w.tx)
}
