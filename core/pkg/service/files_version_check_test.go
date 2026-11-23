// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"encoding/json/v2"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel paths at old versions", Ordered, ContinueOnFailure, func() {
	var (
		l    *service.Layer
		db   *gorp.DB
		proj project.Project
		key  channel.Key
		path string
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		l = MustOpen(svctestutil.OpenLayer(ctx, node))
		folder, name := UniqueChannelName(), UniqueChannelName()
		path = folder + "." + name
		proj = project.Project{Name: "Versions"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := l.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
				return err
			}
			g := group.Group{Name: folder, Scope: resource.TypeChannel}
			if err := l.Group.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			ch := channel.Channel{
				Name:     name,
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    new(g.Key),
			}
			if err := l.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			key = ch.Key()
			return nil
		})).To(Succeed())
	})

	// importFile imports the file body on tx as the API import route does, with one
	// LinkReader for its channel links.
	importFile := func(ctx SpecContext, tx gorp.Tx, body string) resource.ID {
		GinkgoHelper()
		var env imex.Envelope
		Expect(json.Unmarshal([]byte(body), &env)).To(Succeed())
		links := l.Channel.NewLinkReader(ctx, tx, readAll)
		write := MustSucceed(l.ImEx.Decode(
			ctx,
			env.WithDecodeOptions(links.Options()),
			"file.json",
		))
		Expect(links.Err()).To(Succeed())
		return MustSucceed(write(ctx, tx, imex.ImportOptions{
			FileName: "file.json",
			Parent:   project.ResourceID(proj.Key),
		}))
	}
	linkValues := []TableEntry{
		Entry("by key", func() string { return fmt.Sprint(key) }),
		Entry("by path", func() string { return fmt.Sprintf("%q", path) }),
	}

	DescribeTable("Should bind the channel of a v8 schematic", func(
		ctx SpecContext,
		link func() string,
	) {
		tx := DeferClose(db.OpenTx())
		id := importFile(ctx, tx, fmt.Sprintf(`{
			"version": 8,
			"type": "schematic",
			"name": "V8",
			"snapshot": false,
			"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}, "z_index": 0}],
			"edges": [],
			"configs": {"n1": {"variant": "value", "channel": %s}}
		}`, link()))
		var sch schematic.Schematic
		Expect(l.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(uuid.MustParse(id.Key))).
			Entry(&sch).
			Exec(ctx, tx)).To(Succeed())
		Expect(sch.Nodes).To(HaveLen(1))
		value, ok := sch.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(value.Channel).To(Equal(key))
	}, linkValues)

	DescribeTable("Should bind the channel of a v1 table cell", func(
		ctx SpecContext,
		link func() string,
	) {
		tx := DeferClose(db.OpenTx())
		id := importFile(ctx, tx, fmt.Sprintf(`{
			"version": 1,
			"type": "table",
			"name": "V1",
			"rows": [{"size": 30, "cells": ["c1"]}],
			"columns": [{"size": 100}],
			"cells": {"c1": {"key": "c1", "variant": "value", "props": {"channel": %s}}}
		}`, link()))
		var tbl table.Table
		Expect(l.Table.NewRetrieve().
			Where(table.MatchKeys(uuid.MustParse(id.Key))).
			Entry(&tbl).
			Exec(ctx, tx)).To(Succeed())
		Expect(tbl.Rows).To(HaveLen(1))
		Expect(tbl.Rows[0].Cells).To(HaveLen(1))
		value, ok := tbl.Rows[0].Cells[0].Config.Variant.(table.ValueCellConfig)
		Expect(ok).To(BeTrue())
		Expect(value.Channel).To(Equal(key))
	}, linkValues)
})
