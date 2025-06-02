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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
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

var _schema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":    {Type: schema.String},
		"type":   {Type: schema.String},
		"config": {Type: schema.String},
	},
}

func newResource(c Annotation) schema.Resource {
	// Using Type as the display name since annotations don't have a name field
	e := schema.NewResource(_schema, OntologyID(c.Key), "")
	//schema.Set(e, "key", c.Key.String())
	//schema.Set(e, "type", c)
	//schema.Set(e, "config", string(c.Config))
	return e
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, Annotation]

// Schema implements ontology.Service.
func (s *Service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (schema.Resource, error) {
	k := uuid.MustParse(key)
	var annotation Annotation
	err := s.NewRetrieve().WhereKeys(k).Entry(&annotation).Exec(ctx, tx)
	return newResource(annotation), err
}

func translateAnnotationChange(c change) schema.Change {
	return schema.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[schema.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Annotation]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{Wrap: reader, Translate: translateAnnotationChange})
	}
	return gorp.Observe[uuid.UUID, Annotation](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Annotation](s.cfg.DB).OpenNexter()
	return iter.NexterCloserTranslator[Annotation, schema.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
