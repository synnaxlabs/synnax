// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package event

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/service/annotation"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

func newAnnotationCreate(_ context.Context, cfg factoryConfig) (bool, error) {
	if cfg.node.Type != spec.CreateAnnotationType {
		return false, nil
	}
	message, _ := schema.Get[string](schema.Resource{Data: cfg.node.Data}, "message")
	variant, _ := schema.Get[string](schema.Resource{Data: cfg.node.Data}, "variant")
	sink := &confluence.UnarySink[spec.Value]{
		Sink: func(ctx context.Context, value spec.Value) error {
			err := cfg.Annotation.NewWriter(nil).Create(
				ctx,
				&annotation.Annotation{
					Message: message,
					Variant: variant,
				})
			return err
		},
	}
	plumber.SetSink[spec.Value](cfg.pipeline, address.Address(cfg.node.Key), sink)
	return true, nil
}
