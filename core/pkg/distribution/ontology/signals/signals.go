// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"context"
	"io"
	"iter"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// Publish publishes changes from the provided ontology into the provided
// signals.Provider.
func Publish(
	ctx context.Context,
	prov *signals.Provider,
	otg *ontology.Ontology,
) (io.Closer, error) {
	resourceObserver := observe.Translator[iter.Seq[ontology.Change], []change.Change[[]byte, struct{}]]{
		Observable: otg.ResourceObserver,
		Translate: func(ctx context.Context, nexter iter.Seq[ontology.Change]) ([]change.Change[[]byte, struct{}], bool) {
			var (
				out []change.Change[[]byte, struct{}]
				key []byte
				err error
			)
			for ch := range nexter {
				if ch.Variant == change.VariantSet {
					key, err = signals.MarshalJSON(ch.Value)
					if err != nil {
						otg.L.DPanic("unexpected failure to marshal ontology resource set", zap.Error(err))
						continue
					}
				} else {
					key = telem.MarshalVariableSample([]byte(ch.Key))
				}
				out = append(out, change.Change[[]byte, struct{}]{Key: key, Variant: ch.Variant})
			}
			return out, true
		},
	}
	resourceObserverCloser, err := prov.PublishFromObservable(ctx, signals.ObservablePublisherConfig{
		Name:          "ontology_resource",
		Observable:    resourceObserver,
		SetChannel:    channel.Channel{Name: "sy_ontology_resource_set", DataType: telem.JSONT, Internal: true},
		DeleteChannel: channel.Channel{Name: "sy_ontology_resource_delete", DataType: telem.StringT, Internal: true},
	})
	if err != nil {
		return nil, err
	}
	relationshipObserver := observe.Translator[gorp.TxReader[[]byte, ontology.Relationship], []change.Change[[]byte, struct{}]]{
		Observable: otg.RelationshipObserver,
		Translate: func(ctx context.Context, nexter gorp.TxReader[[]byte, ontology.Relationship]) ([]change.Change[[]byte, struct{}], bool) {
			var out []change.Change[[]byte, struct{}]
			for ch := range nexter {
				out = append(out, change.Change[[]byte, struct{}]{
					Key:     telem.MarshalVariableSample(ch.Key),
					Variant: ch.Variant,
				})
			}
			return out, true
		},
	}
	relationshipObserverCloser, err := prov.PublishFromObservable(ctx, signals.ObservablePublisherConfig{
		Name:          "ontology_relationship",
		Observable:    relationshipObserver,
		SetChannel:    channel.Channel{Name: "sy_ontology_relationship_set", DataType: telem.StringT, Internal: true},
		DeleteChannel: channel.Channel{Name: "sy_ontology_relationship_delete", DataType: telem.StringT, Internal: true},
	})
	if err != nil {
		return nil, err
	}
	return xio.MultiCloser{resourceObserverCloser, relationshipObserverCloser}, nil
}

func EncodeID(id ontology.ID) []byte { return telem.MarshalVariableSample([]byte(id.String())) }

func EncodeIDs(ids []ontology.ID) []byte {
	var buf []byte
	for _, id := range ids {
		buf = append(buf, EncodeID(id)...)
	}
	return buf
}

func DecodeRelationships(ser []byte) ([]ontology.Relationship, error) {
	samples := telem.UnmarshalSeries[string](telem.Series{DataType: telem.StringT, Data: ser})
	relationships := make([]ontology.Relationship, 0, len(samples))
	for _, s := range samples {
		relationship, err := ontology.ParseRelationship([]byte(s))
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, relationship)
	}
	return relationships, nil
}

func DecodeIDs(ser []byte) ([]ontology.ID, error) {
	samples := telem.UnmarshalSeries[string](telem.Series{DataType: telem.StringT, Data: ser})
	ids := make([]ontology.ID, 0, len(samples))
	for _, s := range samples {
		id, err := ontology.ParseID(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
