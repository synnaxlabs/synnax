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
	"os"
	"strings"
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
	"github.com/synnaxlabs/synnax/pkg/service/imex/session"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Console file paths check", Ordered, ContinueOnFailure, func() {
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
		proj = project.Project{Name: "Console files"}
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

	// importFile imports body on tx through one import session, as the API import
	// route does, and returns the key of the record it creates.
	importFile := func(ctx SpecContext, tx gorp.Tx, body string) uuid.UUID {
		GinkgoHelper()
		var env imex.Envelope
		Expect(json.Unmarshal([]byte(body), &env)).To(Succeed())
		imp := session.NewImport(ctx, l.ImEx, l.Channel, tx, readAll)
		write := MustSucceed(imp.Decode(env, "file.json"))
		Expect(imp.Err()).To(Succeed())
		id := MustSucceed(write(ctx, tx, imex.ImportOptions{
			FileName: "file.json",
			Parent:   project.ResourceID(proj.Key),
		}))
		return uuid.MustParse(id.Key)
	}
	linkValues := []TableEntry{
		Entry("by key", func() string { return fmt.Sprint(key) }),
		Entry("by path", func() string { return fmt.Sprintf("%q", path) }),
	}

	DescribeTable("Should bind the channel of a v0 Console table cell", func(
		ctx SpecContext,
		link func() string,
	) {
		tx := DeferClose(db.OpenTx())
		k := importFile(ctx, tx, fmt.Sprintf(`{
			"version": "0.0.0",
			"type": "table",
			"name": "T0",
			"layout": {
				"rows": [{"size": 30, "cells": [{"key": "c1"}]}],
				"columns": [{"size": 100}]
			},
			"cells": {"c1": {
				"key": "c1",
				"variant": "value",
				"selected": false,
				"props": {"channel": %s}
			}}
		}`, link()))
		var tbl table.Table
		Expect(l.Table.NewRetrieve().
			Where(table.MatchKeys(k)).
			Entry(&tbl).
			Exec(ctx, tx)).To(Succeed())
		Expect(tbl.Rows).To(HaveLen(1))
		Expect(tbl.Rows[0].Cells).To(HaveLen(1))
		value, ok := tbl.Rows[0].Cells[0].Config.Variant.(table.ValueCellConfig)
		Expect(ok).To(BeTrue(), "cell is %T", tbl.Rows[0].Cells[0].Config.Variant)
		Expect(value.Channel).To(Equal(key))
	}, linkValues)

	DescribeTable("Should bind the channel of a v6 Console schematic export", func(
		ctx SpecContext,
		link func() string,
	) {
		tx := DeferClose(db.OpenTx())
		k := importFile(ctx, tx, fmt.Sprintf(`{
			"version": "6.0.0",
			"type": "schematic",
			"name": "S6",
			"snapshot": false,
			"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}, "zIndex": 0}],
			"edges": [],
			"configs": {"n1": {"variant": "value", "channel": %s}}
		}`, link()))
		var sch schematic.Schematic
		Expect(l.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(k)).
			Entry(&sch).
			Exec(ctx, tx)).To(Succeed())
		Expect(sch.Nodes).To(HaveLen(1))
		value, ok := sch.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue(), "node is %T", sch.Nodes[0].Config.Variant)
		Expect(value.Channel).To(Equal(key))
	}, linkValues)

	DescribeTable("Should bind the channel of a v2 Console schematic state", func(
		ctx SpecContext,
		link func() string,
	) {
		tx := DeferClose(db.OpenTx())
		k := importFile(ctx, tx, fmt.Sprintf(`{
			"version": "2.0.0",
			"type": "schematic",
			"name": "S2",
			"editable": true,
			"snapshot": false,
			"remoteCreated": false,
			"viewport": {"position": {"x": 0, "y": 0}, "zoom": 1},
			"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}, "zIndex": 1}],
			"edges": [],
			"props": {"n1": {
				"key": "value",
				"units": "psi",
				"telem": {
					"type": "source-pipeline",
					"variant": "source",
					"valueType": "string",
					"props": {
						"outlet": "stringifier",
						"connections": [
							{"from": "valueStream", "to": "rollingAverage"},
							{"from": "rollingAverage", "to": "stringifier"}
						],
						"segments": {
							"valueStream": {
								"type": "stream-channel-value",
								"variant": "source",
								"valueType": "number",
								"props": {"channel": %s}
							},
							"rollingAverage": {
								"type": "rolling-average",
								"variant": "source",
								"valueType": "number",
								"props": {"windowSize": 1}
							},
							"stringifier": {
								"type": "stringify-number",
								"variant": "source",
								"valueType": "string",
								"props": {"precision": 2}
							}
						}
					}
				}
			}}
		}`, link()))
		var sch schematic.Schematic
		Expect(l.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(k)).
			Entry(&sch).
			Exec(ctx, tx)).To(Succeed())
		Expect(sch.Nodes).To(HaveLen(1))
		value, ok := sch.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue(), "node is %T", sch.Nodes[0].Config.Variant)
		Expect(value.Channel).To(Equal(key))
	}, linkValues)

	DescribeTable("Should bind the channel of a v1 Console log", func(
		ctx SpecContext,
		link func() string,
	) {
		tx := DeferClose(db.OpenTx())
		k := importFile(ctx, tx, fmt.Sprintf(`{
			"version": "1.0.0",
			"type": "log",
			"name": "L1",
			"channels": [{
				"channel": %s,
				"color": "",
				"notation": "standard",
				"precision": -1,
				"alias": ""
			}],
			"remoteCreated": false,
			"timestampPrecision": 0,
			"showChannelNames": true,
			"showReceiptTimestamp": false
		}`, link()))
		var lg log.Log
		Expect(l.Log.NewRetrieve().
			Where(log.MatchKeys(k)).
			Entry(&lg).
			Exec(ctx, tx)).To(Succeed())
		Expect(lg.Channels).To(HaveLen(1))
		Expect(lg.Channels[0].Channel).To(Equal(key))
	}, linkValues)

	DescribeTable("Should bind the channel of a v1 Console line plot", func(
		ctx SpecContext,
		link func() string,
	) {
		tx := DeferClose(db.OpenTx())
		b := string(MustSucceed(
			os.ReadFile("lineplot/versions/testdata/import_v1_state.json"),
		))
		const y1 = `"y1": [65541, 65542]`
		Expect(b).To(ContainSubstring(y1))
		b = strings.Replace(b, y1, fmt.Sprintf(`"y1": [%s]`, link()), 1)
		k := importFile(ctx, tx, b)
		var lp lineplot.LinePlot
		Expect(l.LinePlot.NewRetrieve().
			Where(lineplot.MatchKeys(k)).
			Entry(&lp).
			Exec(ctx, tx)).To(Succeed())
		Expect(lp.Channels.Y1).To(Equal([]channel.Key{key}))
	}, linkValues)
})
