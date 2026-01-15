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

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

// Writer is used to create, update, and delete devices within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly on
// the database.
type Writer struct {
	tx     gorp.Tx
	otg    ontology.Writer
	group  group.Group
	status status.Writer[StatusDetails]
}

func resolveStatus(d *Device, provided *Status) *Status {
	if provided == nil {
		return &Status{
			Key:     OntologyID(d.Key).String(),
			Name:    d.Name,
			Time:    telem.Now(),
			Variant: xstatus.VariantWarning,
			Message: fmt.Sprintf("%s state unknown", d.Name),
			Details: StatusDetails{Rack: d.Rack, Device: d.Key},
		}
	}
	provided.Key = OntologyID(d.Key).String()
	provided.Name = d.Name
	provided.Details.Device = d.Key
	provided.Details.Rack = d.Rack
	return provided
}

// Create creates or updates the given device. Create will redefine ontology
// relationships in the ontology if the device has moved racks. If a status is
// provided on the device, it will be used instead of the default "unknown" status.
func (w Writer) Create(ctx context.Context, device Device) error {
	if err := device.Validate(); err != nil {
		return err
	}
	providedStatus := device.Status // Preserve before clearing for gorp
	device.Status = nil             // Status stored separately, not in gorp
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
	// If the device already exists with the same rack and parent, don't redefine
	// the resource and relationship in the ontology, as to not mess with existing
	// groups or relationships.
	if exists && device.Rack == existing.Rack && device.ParentDevice == existing.ParentDevice {
		// If the device is being renamed, update the status name.
		if device.Name != existing.Name {
			stat := resolveStatus(&device, providedStatus)
			return w.status.Set(ctx, stat)
		}
		return nil
	}
	stat := resolveStatus(&device, providedStatus)
	if err = w.status.Set(ctx, stat); err != nil {
		return err
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
	// If the device has a parent device, create the relationship to the parent device.
	// Otherwise, create the relationship to the rack.
	var parentOntologyID ontology.ID
	if device.ParentDevice != "" {
		parentOntologyID = OntologyID(device.ParentDevice)
	} else {
		parentOntologyID = device.Rack.OntologyID()
	}
	return w.otg.DefineRelationship(
		ctx,
		parentOntologyID,
		ontology.ParentOf,
		otgID,
	)
}

// Delete deletes the device with the given key and its associated status.
func (w Writer) Delete(ctx context.Context, key string) error {
	if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
		return err
	}
	if err := w.status.Delete(ctx, OntologyID(key).String()); err != nil {
		return err
	}
	return gorp.NewDelete[string, Device]().WhereKeys(key).Exec(ctx, w.tx)
}
