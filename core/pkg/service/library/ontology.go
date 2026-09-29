// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"bytes"
	"context"
	"io"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// RelationshipTypeUses indicates that a resource reads a library. The From field is the
// resource, and the To field is the library it uses.
const RelationshipTypeUses ontology.RelationshipType = "uses"

var relationshipTypeUsesBytes = []byte(RelationshipTypeUses)

// UsersTraverser traverses from libraries to the resources that use them.
var UsersTraverser = ontology.Traverser{
	Traverse: func(ids []ontology.ID) ontology.RawTraversal {
		w := orc.NewWriter(64)
		encoded := make([][]byte, len(ids))
		for i, id := range ids {
			w.Reset()
			w.String(string(id.Type))
			w.String(id.Key)
			encoded[i] = w.Copy()
		}
		return func(data []byte, nextIDs *[]ontology.ID) error {
			raw, err := orc.NewRaw(data)
			if err != nil {
				return err
			}
			fromType, r := raw.ReadString()
			fromKey, r := r.ReadString()
			relType, r := r.ReadString()
			if !bytes.Equal(relType, relationshipTypeUsesBytes) {
				return nil
			}
			for _, enc := range encoded {
				if bytes.HasPrefix(r, enc) {
					*nextIDs = append(*nextIDs, ontology.ID{
						Type: ontology.ResourceType(fromType),
						Key:  string(fromKey),
					})
				}
			}
			return nil
		}
	},
	Direction: ontology.DirectionBackward,
}

// OntologyID returns the ontology ID of the library with the given key.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeLibrary, Key: key.String()}
}

// OntologyIDs returns the ontology IDs of the libraries with the given keys.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromLibraries returns the ontology IDs of the given libraries.
func OntologyIDsFromLibraries(libs []Library) []ontology.ID {
	return lo.Map(libs, func(l Library, _ int) ontology.ID { return l.OntologyID() })
}

// KeysFromOntologyIDs returns the library keys of the given ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id ontology.ID, _ int) (Key, error) {
		return uuid.Parse(id.Key)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

func newResource(l Library) ontology.Resource {
	return ontology.NewResource(schema, l.OntologyID(), l.Name, l)
}

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

type change = xchange.Change[Key, Library]

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeLibrary }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var l Library
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&l).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(l), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Library]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
