// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/taskreporter"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

const deleteMemberName = "delete"

var deleteParams = types.Params{
	{Name: "key_or_name", Type: types.String()},
}

var deleteSymbolProps = types.Function(types.FunctionProperties{
	Inputs:  deleteParams,
	Config:  deleteParams,
	Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
})

var deleteResolverEntry = symbol.Symbol{
	Name: deleteMemberName,
	Kind: symbol.KindFunction,
	Exec: symbol.ExecBoth,
	Type: deleteSymbolProps,
}

type deleteNode struct {
	*node.State
	stat      *status.Service
	ins       alamos.Instrumentation
	report    taskreporter.Reporter
	keyOrName string
}

func (s *deleteNode) Next(ctx node.Context) {
	var v uint8
	if dispatchDelete(ctx, s.stat, s.ins, s.report, s.keyOrName) {
		v = 1
	}
	*s.Output(0) = telem.NewSeriesV[uint8](v)
	*s.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())
	ctx.MarkChanged(0)
}

// dispatchDelete deletes a status by key (UUID) or by name. Returns true if at
// least one row was deleted. Reports warnings on not-found, multi-match
// (deletes all), or other failures.
func dispatchDelete(
	ctx context.Context,
	stat *status.Service,
	ins alamos.Instrumentation,
	report taskreporter.Reporter,
	keyOrName string,
) bool {
	count, err := stat.DeleteByKeyOrName(ctx, keyOrName)
	if err != nil {
		ins.L.Error("status.delete failed", zap.String("key_or_name", keyOrName), zap.Error(err))
		report(ctx, xstatus.VariantWarning, fmt.Sprintf("status.delete: %v", err))
		return false
	}
	if count == 0 {
		report(ctx, xstatus.VariantWarning, fmt.Sprintf("status.delete: no status found %q", keyOrName))
		return false
	}
	if count > 1 {
		report(ctx, xstatus.VariantWarning,
			fmt.Sprintf("status.delete: multiple statuses named %q; deleted all (%d)", keyOrName, count))
	}
	return true
}

