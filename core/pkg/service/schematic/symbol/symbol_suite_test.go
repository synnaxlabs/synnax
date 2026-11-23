// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	namedcolor "github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestSymbol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Schematic Symbol Suite", SuiteConfig())
}

var (
	db       *gorp.DB
	proj     project.Project
	svc      *symbol.Service
	imexSvc  *imex.Service
	groupSvc *group.Service
	enumSvc  *enumeration.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(db.Run)
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	imexSvc = imex.NewService()
	projectSvc := MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB: db,
	}))
	enumSvc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
	svc = MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
		DB:          db,
		Group:       groupSvc,
		ImEx:        imexSvc,
		Enumeration: enumSvc,
		Actuators:   noActuators,
	}))
	proj.Name = "test-project"
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
})

var _ = ShouldNotLeakGoroutinesPerSpec()

// svg returns a drawing of markup.
func svg(markup string) symbol.Drawing {
	return symbol.Drawing{Variant: symbol.SvgDrawing{Markup: markup}}
}

// colorPaint returns a paint of the color that hex names.
func colorPaint(hex string) symbol.Paint {
	GinkgoHelper()
	return symbol.Paint{Variant: symbol.ColorPaint{Value: namedcolor.Ref{
		Variant: namedcolor.FixedRef{Value: MustSucceed(color.Parse(hex))},
	}}}
}

// noActuators reports that no custom_actuator node draws a symbol.
func noActuators(context.Context, gorp.Tx) (set.Set[symbol.Key], error) {
	return nil, nil
}
