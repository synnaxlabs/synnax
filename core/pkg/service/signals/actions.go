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
	"fmt"
	"io"

	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/validate"
)

// ActionPublisherConfig configures PublishActions.
type ActionPublisherConfig[K comparable, A any] struct {
	// Observable emits the action sequences to publish, such as an actions.State.
	//
	// [REQUIRED]
	Observable observe.Observable[actions.Scoped[K, A]]
	// Name is the service name segment of the channel name ("sy_<name>_set") and the
	// pipeline name ("<name>_actions").
	//
	// [REQUIRED]
	Name string
}

// Validate implements config.Config.
func (c ActionPublisherConfig[K, A]) Validate() error {
	v := validate.New("signals.action_publisher_config")
	v.NotNil("observable", c.Observable)
	v.NotEmptyString("name", c.Name)
	return v.Error()
}

// PublishActions opens a pipeline that broadcasts each action sequence the observable
// emits on the sy_<name>_set free channel as a JSON sample. The returned io.Closer
// shuts the pipeline down.
func (p *Provider) PublishActions[K comparable, A any](
	ctx context.Context,
	cfg ActionPublisherConfig[K, A],
) (io.Closer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	closer, err := p.PublishJSON(ctx, JSONPublisherConfig[actions.Scoped[K, A]]{
		Name:       fmt.Sprintf("%s_actions", cfg.Name),
		Observable: cfg.Observable,
		SetName:    fmt.Sprintf("sy_%s_set", cfg.Name),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "open action publisher for %s", cfg.Name)
	}
	return closer, nil
}
