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
	"io"
	"iter"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyType is a unique type for the device service within the ontology.
const OntologyType ontology.Type = "device"

// OntologyID returns the unique ID for the device within the ontology.
func OntologyID(key string) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: key}
}

// OntologyIDsFromDevices returns the ontology IDs for the given devices.
func OntologyIDsFromDevices(devices []Device) []ontology.ID {
	return lo.Map(devices, func(d Device, _ int) ontology.ID {
		return OntologyID(d.Key)
	})
}

// OntologyIDs returns the ontology IDs for the given keys.
func OntologyIDs(keys []string) []ontology.ID {
	return lo.Map(keys, func(k string, _ int) ontology.ID { return OntologyID(k) })
}

// KeysFromOntologyIDs returns the keys for the given ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) []string {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = id.Key
	}
	return keys
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":          zyn.String(),
	"rack":         zyn.Uint32().Coerce(),
	"location":     zyn.String(),
	"name":         zyn.String(),
	"make":         zyn.String(),
	"model":        zyn.String(),
	"configured":   zyn.Bool(),
	"has_children": zyn.Bool(),
})

// resourceData wraps Device with a HasChildren field derived from actual ontology
// relationships, allowing the Console to determine expandability without extra API calls.
type resourceData struct {
	Device
	HasChildren bool `json:"has_children"`
}

func newResource(d Device, hasChildren bool) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(d.Key), d.Name, resourceData{
		Device:      d,
		HasChildren: hasChildren,
	})
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[string, Device]

// Type returns the type of the device ontology service.
func (s *Service) Type() ontology.Type { return OntologyType }

// Schema returns the schema for the device ontology service.
func (s *Service) Schema() zyn.Schema { return schema }

// SearchableFields implements ontology.SearchableFieldsProvider.
func (s *Service) SearchableFields() []string {
	return []string{"make", "model", "location"}
}

// RetrieveResource allows for retrieving a device with a given key from the ontology.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	var d Device
	if err := s.NewRetrieve().WhereKeys(key).Entry(&d).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	var children []ontology.Resource
	_ = s.cfg.Ontology.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ChildrenTraverser).
		Limit(1).
		ExcludeFieldData(true).
		Entries(&children).
		Exec(ctx, tx)
	return newResource(d, len(children) > 0), nil
}

func (s *Service) translateChange(ctx context.Context, c change) ontology.Change {
	hasChildren := false
	if c.Variant != xchange.VariantDelete {
		var children []ontology.Resource
		_ = s.cfg.Ontology.NewRetrieve().
			WhereIDs(OntologyID(c.Key)).
			TraverseTo(ontology.ChildrenTraverser).
			Limit(1).
			ExcludeFieldData(true).
			Entries(&children).
			Exec(ctx, nil)
		hasChildren = len(children) > 0
	}
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value, hasChildren),
	}
}

// OnChange implements determines what should happen in the ontology when a change is
// made to a device.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[string, Device]) {
		f(ctx, xiter.Map(reader, func(c change) ontology.Change {
			return s.translateChange(ctx, c)
		}))
	}
	return gorp.Observe[string, Device](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter opens a nexter type that allows for iterating over all devices in the
// ontology.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[string, Device](s.cfg.DB).OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, func(d Device) ontology.Resource {
		return newResource(d, false)
	}), closer, nil
}
