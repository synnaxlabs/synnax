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
	"encoding/json"
	"io"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// JSONPublisherConfig is the configuration for opening a Signals pipeline that writes
// every value an observable emits to a single channel as one JSON sample.
type JSONPublisherConfig[V any] struct {
	// Observable emits the values to publish.
	//
	// [REQUIRED]
	Observable observe.Observable[V]
	// SetName is the name of the channel the values are written to.
	//
	// [REQUIRED]
	SetName string
	// Name identifies the pipeline for debugging purposes.
	//
	// [OPTIONAL] - Defaults to SetName.
	Name string
}

var _ config.Config[JSONPublisherConfig[any]] = JSONPublisherConfig[any]{}

// Override implements config.Config.
func (c JSONPublisherConfig[V]) Override(
	other JSONPublisherConfig[V],
) JSONPublisherConfig[V] {
	c.Observable = override.Nil(c.Observable, other.Observable)
	c.SetName = override.String(c.SetName, other.SetName)
	c.Name = override.String(c.Name, other.Name)
	if c.Name == "" {
		c.Name = c.SetName
	}
	return c
}

// Validate implements config.Config.
func (c JSONPublisherConfig[V]) Validate() error {
	v := validate.New("signals.json_publisher_config")
	v.NotNil("observable", c.Observable)
	v.NotEmptyString("set_name", c.SetName)
	return v.Error()
}

// PublishJSON opens a Signals pipeline that writes every value the configured
// observable emits to the configured channel as one JSON sample. A value that fails to
// marshal is logged and dropped. The returned io.Closer stops the pipeline.
func (p *Provider) PublishJSON[V any](
	ctx context.Context,
	cfgs ...JSONPublisherConfig[V],
) (io.Closer, error) {
	cfg, err := config.New(JSONPublisherConfig[V]{}, cfgs...)
	if err != nil {
		return nil, err
	}
	obs := observe.Translator[V, []change.Change[[]byte, struct{}]]{
		Observable: cfg.Observable,
		Translate: func(_ context.Context, v V) ([]change.Change[[]byte, struct{}], bool) {
			b, err := json.Marshal(v)
			if err != nil {
				p.cfg.L.Error(
					"failed to marshal value",
					zap.Error(err),
					zap.String("channel", cfg.SetName),
				)
				return nil, false
			}
			return []change.Change[[]byte, struct{}]{{
				Variant: change.VariantSet,
				Key:     telem.MarshalVariableSample(b),
			}}, true
		},
	}
	return p.PublishFromObservable(ctx, ObservablePublisherConfig{
		Name:       cfg.Name,
		Observable: obs,
		SetChannel: channel.Channel{
			Name:     cfg.SetName,
			DataType: telem.JSONT,
			Internal: true,
		},
	})
}
