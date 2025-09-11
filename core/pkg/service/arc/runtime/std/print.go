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
	"fmt"

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
)

var symbolPrinter = arc.Symbol{
	Name: "printer",
	Kind: ir.KindStage,
	Type: ir.Stage{Key: "printer"},
}

type printer struct{ base }

func createPrinter(_ context.Context, cfg Config) (stage.Stage, error) {
	return &printer{base{key: cfg.Node.Key}}, nil
}

func (p *printer) Next(_ context.Context, value stage.Value) {
	fmt.Println(value)
}
