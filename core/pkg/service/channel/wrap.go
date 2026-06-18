// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	"github.com/synnaxlabs/x/telem"
)

// WrapOption configures the channel Service opened by Wrap.
type WrapOption func(*wrapOptions)

type wrapOptions struct {
	intOverflowCheck IntOverflowChecker
	validateNames    *bool
}

// WithIntOverflowCheck overrides the overflow checker Wrap installs on the channel
// Service. When unset, Wrap defaults to the free-tier check (verification.FreeOverflowCheck).
func WithIntOverflowCheck(check IntOverflowChecker) WrapOption {
	return func(o *wrapOptions) { o.intOverflowCheck = check }
}

// WithValidateNames overrides whether the channel Service validates channel names. When
// unset, Wrap defaults to enabling validation.
func WithValidateNames(validate bool) WrapOption {
	return func(o *wrapOptions) { o.validateNames = &validate }
}

// Wrap opens a channel Service on top of the provided distribution layer, binds it as the
// layer's channel retriever (so the distribution framer can resolve channels), and
// performs the same node-control-channel setup the service layer does in production. It
// is a convenience for tests and other lightweight contexts that already have a fully
// opened distribution layer; production code should use NewService with an explicit
// ServiceConfig and perform control-channel setup in the service layer. The overflow
// check defaults to the free tier unless overridden with WithIntOverflowCheck. Wrap
// panics if the service cannot be opened or the control channel cannot be configured.
func Wrap(dist *distribution.Layer, opts ...WrapOption) *Service {
	o := wrapOptions{intOverflowCheck: verification.FreeOverflowCheck, validateNames: new(true)}
	for _, opt := range opts {
		opt(&o)
	}
	s, err := NewService(context.Background(), ServiceConfig{
		DB:               dist.DB,
		Allocator:        dist.Channel,
		HostResolver:     dist.Cluster,
		Ontology:         dist.Ontology,
		Group:            dist.Group,
		Search:           dist.Search,
		IntOverflowCheck: o.intOverflowCheck,
		ValidateNames:    o.validateNames,
	})
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	controlCh := Channel{
		Name:        fmt.Sprintf("sy_node_%v_control", dist.Cluster.HostKey()),
		Leaseholder: dist.Cluster.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}
	if err := s.Create(ctx, &controlCh, RetrieveIfNameExists()); err != nil {
		panic(err)
	}
	if err := dist.Framer.ConfigureControlUpdateChannel(ctx, controlCh.Key(), controlCh.Name); err != nil {
		panic(err)
	}
	return s
}
