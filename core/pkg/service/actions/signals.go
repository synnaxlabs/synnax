// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// SignalsConfig configures PublishSignals.
type SignalsConfig[K comparable, A any] struct {
	// Provider opens the underlying signals pipeline.
	//
	// [REQUIRED]
	Provider *signals.Provider
	// State supplies the action observable broadcast on the set channel.
	//
	// [REQUIRED]
	State *State[K, A]
	// Name is the service name segment used to build the channel name
	// ("sy_<name>_set") and the pipeline name ("<name>_actions"). Pass
	// the same string the service uses for its other signals channels so
	// callers can locate the action stream by convention.
	//
	// [REQUIRED]
	Name string
}

// Validate implements config.Config.
func (c SignalsConfig[K, A]) Validate() error {
	v := validate.New("actions.signals_config")
	validate.NotNil(v, "provider", c.Provider)
	validate.NotNil(v, "state", c.State)
	validate.NotEmptyString(v, "name", c.Name)
	return v.Error()
}

// PublishSignals opens a cluster signals pipeline that broadcasts every Scoped
// envelope emitted by cfg.State on the sy_<name>_set free channel as
// JSON-encoded variable samples. The returned io.Closer shuts the pipeline
// down. Entry-level set and delete signaling is a separate concern owned by
// the per-service OpenService and is not handled here.
func PublishSignals[K comparable, A any](
	ctx context.Context,
	cfg SignalsConfig[K, A],
) (io.Closer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	translator := observe.Translator[Scoped[K, A], []xchange.Change[[]byte, struct{}]]{
		Observable: cfg.State.observer,
		Translate: func(_ context.Context, sa Scoped[K, A]) ([]xchange.Change[[]byte, struct{}], bool) {
			b, err := json.Marshal(sa)
			if err != nil {
				return nil, false
			}
			return []xchange.Change[[]byte, struct{}]{
				{Variant: xchange.VariantSet, Key: telem.MarshalVariableSample(b)},
			}, true
		},
	}
	closer, err := cfg.Provider.PublishFromObservable(ctx, signals.ObservablePublisherConfig{
		Name:       fmt.Sprintf("%s_actions", cfg.Name),
		Observable: translator,
		SetChannel: channel.Channel{
			Name:     fmt.Sprintf("sy_%s_set", cfg.Name),
			DataType: telem.JSONT,
			Internal: true,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "open action publisher for %s", cfg.Name)
	}
	return closer, nil
}
