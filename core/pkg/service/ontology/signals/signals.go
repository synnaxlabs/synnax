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
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/change"
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
	resourceObserver := observe.Translator[
		iter.Seq[ontology.Change], []change.Change[[]byte, struct{}],
	]{
		Observable: otg.ResourceObserver,
		Translate: func(
			_ context.Context,
			nexter iter.Seq[ontology.Change],
		) ([]change.Change[[]byte, struct{}], bool) {
			var (
				out []change.Change[[]byte, struct{}]
				key []byte
				err error
			)
			for ch := range nexter {
				if ch.Variant == change.VariantSet {
					key, err = signals.MarshalJSON(ch.Value)
					if err != nil {
						otg.L.DPanic(
							"unexpected failure to marshal ontology resource set",
							zap.Error(err),
						)
						continue
					}
				} else {
					key = telem.MarshalVariableSample([]byte(ch.Key))
				}
				out = append(out, change.Change[[]byte, struct{}]{
					Key:     key,
					Variant: ch.Variant,
				})
			}
			return out, true
		},
	}
	resourceCloser, err := prov.PublishFromObservable(
		ctx,
		signals.ObservablePublisherConfig{
			Name:       "ontology_resource",
			Observable: resourceObserver,
			SetChannel: channel.Channel{
				Name:     "sy_ontology_resource_set",
				DataType: telem.JSONT,
				Internal: true,
			},
			DeleteChannel: channel.Channel{
				Name:     "sy_ontology_resource_delete",
				DataType: telem.StringT,
				Internal: true,
			},
		})
	if err != nil {
		return nil, err
	}
	relationshipCloser, err := signals.PublishFromGorp(
		ctx,
		prov,
		signals.GorpPublisherConfig[string, ontology.Relationship]{
			Observable:     otg.RelationshipObserver,
			SetName:        "sy_ontology_relationship_set",
			DeleteName:     "sy_ontology_relationship_delete",
			SetDataType:    telem.StringT,
			DeleteDataType: telem.StringT,
			MarshalSet: func(r ontology.Relationship) ([]byte, error) {
				return telem.MarshalVariableSample([]byte(r.GorpKey())), nil
			},
			MarshalDelete: func(k string) ([]byte, error) {
				return telem.MarshalVariableSample([]byte(k)), nil
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return xio.MultiCloser{resourceCloser, relationshipCloser}, nil
}
