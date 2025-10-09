// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

type Constructor = func(ctx context.Context, cfg Config) (stage.Stage, error)

var factories = map[string]Constructor{
	"ge":         GEFactory,
	"gt":         GTFactory,
	"le":         LEFactory,
	"lt":         LTFactory,
	"eq":         EQFactory,
	"ne":         NEFactory,
	"add":        AddFactory,
	"sub":        SubFactory,
	"mul":        MulFactory,
	"div":        DivFactory,
	"mod":        ModFactory,
	"constant":   newConstant,
	"select":     createSelect,
	"on":         createChannelSource,
	"stable_for": createStableFor,
	"set_status": createSetStatus,
	"write":      createChannelSink,
}

var Resolver = ir.MapResolver{
	"ge":         symbolGE,
	"gt":         symbolGT,
	"le":         symbolLE,
	"lt":         symbolLT,
	"eq":         symbolEQ,
	"ne":         symbolNE,
	"add":        symbolAdd,
	"sub":        symbolSub,
	"mul":        symbolMul,
	"div":        symbolDiv,
	"mod":        symbolMod,
	"constant":   symbolConstant,
	"on":         symbolChannelSource,
	"write":      symbolChannelSink,
	"select":     symbolSelect,
	"stable_for": symbolStableFor,
	"set_status": symbolSetStatus,
}

type ChannelData interface {
	Get(key channel.Key) telem.Series
}

type Config struct {
	alamos.Instrumentation
	Node        ir.Node
	Status      *status.Service
	ChannelData ChannelData
	Now         func() telem.TimeStamp
	Write       func(ctx context.Context, fr core.Frame) error
	Channel     channel.Readable
}

func Create(ctx context.Context, cfg Config) (stage.Stage, error) {
	v, ok := factories[cfg.Node.Type]
	if !ok {
		return nil, errors.Wrapf(query.NotFound, "std. lib stage with type %s not found", cfg.Node.Type)
	}
	return v(ctx, cfg)
}
