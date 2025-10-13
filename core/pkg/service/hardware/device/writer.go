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

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Writer is used to create, update, and delete devices within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly on
// the database.
type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
}

// Create creates or updates the given device. Create will redefine ontology
// relationships in the ontology if the device has moved racks.
func (w Writer) Create(ctx context.Context, device Device) error {
	if err := device.Validate(); err != nil {
		return err
	}
	var existing Device
	err := gorp.
		NewRetrieve[string, Device]().
		WhereKeys(device.Key).
		Entry(&existing).
		Exec(ctx, w.tx)
	isNotFound := errors.Is(err, query.NotFound)
	if err != nil && !isNotFound {
		return err
	}
	exists := !isNotFound
	if err = gorp.
		NewCreate[string, Device]().
		Entry(&device).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	// If the device already exists, don't redefine the resource and relationship in the
	// ontology, as to not mess with existing groups or relationships.
	if exists && device.Rack == existing.Rack {
		return nil
	}
	otgID := OntologyID(device.Key)
	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if err = w.otg.DeleteIncomingRelationshipsOfType(
		ctx,
		otgID,
		ontology.ParentOf,
	); err != nil {
		return err
	}
	return w.otg.DefineRelationship(
		ctx,
		device.Rack.OntologyID(),
		ontology.ParentOf,
		otgID,
	)
}

// Delete deletes the device with the given key.
func (w Writer) Delete(ctx context.Context, key string) error {
	if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
		return err
	}
	return gorp.NewDelete[string, Device]().WhereKeys(key).Exec(ctx, w.tx)
}
