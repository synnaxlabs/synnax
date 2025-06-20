// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package annotation

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/zyn"
)

const ontologyType ontology.Type = "annotation"

// OntologyID returns a unique identifier for the annotation within the ontology.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the annotations within the ontology.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(key uuid.UUID, _ int) ontology.ID {
		return OntologyID(key)
	})
}

// KeysFromOntologyIDs extracts the keys of the annotations from the ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) (keys []uuid.UUID, err error) {
	keys = make([]uuid.UUID, len(ids))
	for i, id := range ids {
		keys[i], err = uuid.Parse(id.Key)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// OntologyIDsFromAnnotations returns the ontology IDs of the annotations.
func OntologyIDsFromAnnotations(annotations []Annotation) []ontology.ID {
	return lo.Map(annotations, func(c Annotation, _ int) ontology.ID {
		return OntologyID(c.Key)
	})
}

var Z = zyn.Object(map[string]zyn.Z{
	"key":     zyn.UUID(),
	"variant": status.VariantZ,
	"message": zyn.String(),
})

func newResource(c Annotation) core.Resource {
	return core.NewResource(Z, OntologyID(c.Key), c.Message, c)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, Annotation]

// Type implements ontology.Service.
func (s *Service) Type() ontology.Type { return ontologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Z { return Z }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (core.Resource, error) {
	k := uuid.MustParse(key)
	var annotation Annotation
	err := s.NewRetrieve().WhereKeys(k).Entry(&annotation).Exec(ctx, tx)
	return newResource(annotation), err
}

func translateAnnotationChange(c change) core.Change {
	return core.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[core.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Annotation]) {
		f(ctx, iter.NexterTranslator[change, core.Change]{Wrap: reader, Translate: translateAnnotationChange})
	}
	return gorp.Observe[uuid.UUID, Annotation](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[core.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Annotation](s.cfg.DB).OpenNexter()
	return iter.NexterCloserTranslator[Annotation, core.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
