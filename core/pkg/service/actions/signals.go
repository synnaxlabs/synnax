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
	"fmt"
	"io"

	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/errors"
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
	v.NotNil("provider", c.Provider)
	v.NotNil("state", c.State)
	v.NotEmptyString("name", c.Name)
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
	closer, err := cfg.Provider.PublishJSON(
		ctx,
		signals.JSONPublisherConfig[Scoped[K, A]]{
			Name:       fmt.Sprintf("%s_actions", cfg.Name),
			Observable: cfg.State.observer,
			SetName:    fmt.Sprintf("sy_%s_set", cfg.Name),
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "open action publisher for %s", cfg.Name)
	}
	return closer, nil
}
