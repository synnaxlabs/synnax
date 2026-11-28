// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSchematic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Schematic Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	enumSvc    *enumeration.Service
	groupSvc   *group.Service
	colorSvc   *color.Service
	proj       project.Project
	projectSvc *project.Service
	svc        *schematic.Service
	imexSvc    *imex.Service
	channelSvc *channel.Service
	// ch1 to ch3 are channels that the specs bind symbols to.
	ch1, ch2, ch3 channel.Key
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 3))
	ch1, ch2, ch3 = keys[0], keys[1], keys[2]
	enumSvc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{
		DB: db,
	}))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	colorSvc = MustOpen(color.OpenService(ctx, color.ServiceConfig{
		DB:    db,
		Group: groupSvc,
	}))
	imexSvc = imex.NewService()
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB: db,
	}))
	svc = MustOpen(schematic.OpenService(ctx, schematic.ServiceConfig{
		DB:          db,
		Group:       groupSvc,
		ImEx:        imexSvc,
		Enumeration: enumSvc,
	}))
	proj.Name = "test-project"
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
})

// dispatch runs Writer.Apply in a transaction of its own.
func dispatch(
	ctx context.Context,
	key schematic.Key,
	dispatchKey string,
	acts []schematic.Action,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := svc.NewWriter(tx).
			WithDispatchKey(dispatchKey).
			Apply(ctx, key, acts...)
		return err
	})
}

// fixedColor returns a fixed color reference to the color that hex names.
func fixedColor(hex string) *color.Ref {
	GinkgoHelper()
	return &color.Ref{Variant: color.FixedRef{Value: MustSucceed(xcolor.Parse(hex))}}
}
