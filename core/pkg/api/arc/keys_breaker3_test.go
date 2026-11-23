// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rack fetch breaker 3", Ordered, ContinueOnFailure, func() {
	var in, out channel.Channel
	BeforeAll(func(ctx SpecContext) {
		in = channel.Channel{Name: "bk3_in", DataType: telem.Float64T, Virtual: true}
		out = channel.Channel{Name: "bk3_out", DataType: telem.Float64T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			for _, ch := range []*channel.Channel{&in, &out} {
				if err := w.Create(ctx, ch); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
	})
	create := func(ctx SpecContext, b arc.Base) arc.Arc {
		GinkgoHelper()
		a := arc.Arc{Variant: arc.AutomationArc{Base: b}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		key := a.Base().Key
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
		})
		grantOn(ctx, author.ResourceID(), access.ActionRetrieve, a.ResourceID())
		return a
	}
	textBase := func(name, raw string) arc.Base {
		return arc.Base{
			Name:   name,
			Source: arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}},
		}
	}
	graphBase := func(name, sourceType string, source, sink any) arc.Base {
		return arc.Base{
			Name: name,
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "source"}, {Key: "sink"}},
				Edges: graph.Edges{{
					Key:    "edge",
					Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
				}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {"type": sourceType, "channel": source},
					"sink":   {"type": "write", "channel": sink},
				},
			}}},
		}
	}
	request := func(key arc.Key) apiarc.RetrieveRequest {
		return apiarc.RetrieveRequest{
			Where:          &arc.Where{Key: &filter.Set[arc.Key]{In: []arc.Key{key}}},
			IncludeProgram: true,
		}
	}
	fetch := func(
		ctx context.Context,
		key arc.Key,
	) (user, rack apiarc.RetrieveResponse, userErr, rackErr error) {
		user, userErr = retrieve(AuthedCtx(ctx, author), request(key))
		rack, rackErr = retrieve(SubjectCtx(ctx, testRack.ResourceID()), request(key))
		return user, rack, userErr, rackErr
	}
	describe := func(err error) string {
		if err == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%T: %s", err, err.Error())
	}

	It("Should fail a rack's fetch of a text automation that does not analyze with "+
		"the error of a user's fetch", func(ctx SpecContext) {
		a := create(ctx, textBase("bk3_bad_text", "bk3_missing * 2.0 -> bk3_out"))
		deploy(ctx, a)
		_, _, userErr, rackErr := fetch(ctx, a.Base().Key)
		Expect(userErr).To(HaveOccurred())
		Expect(describe(rackErr)).To(Equal(describe(userErr)))
	})

	It("Should fail a rack's fetch of a graph automation that does not analyze with "+
		"the error of a user's fetch", func(ctx SpecContext) {
		a := create(ctx, graphBase(
			"bk3_bad_graph",
			"bk3_no_such_function",
			float64(in.Key()),
			float64(out.Key()),
		))
		deploy(ctx, a)
		_, _, userErr, rackErr := fetch(ctx, a.Base().Key)
		Expect(userErr).To(HaveOccurred())
		Expect(rackErr).To(HaveOccurred())
		// The texts agree. The rack gets a CompileError, as a text compile gives, and a
		// user gets the raw diagnostics of the graph compile.
		Expect(rackErr.Error()).To(Equal(userErr.Error()))
		Expect(rackErr).To(BeAssignableToTypeOf(apiarc.CompileError{}))
	})

	It("Should fail a rack's fetch of a graph whose node input is no key with the "+
		"error of a user's fetch", func(ctx SpecContext) {
		a := create(ctx, graphBase("bk3_bad_input", "on", true, float64(out.Key())))
		deploy(ctx, a)
		_, _, userErr, rackErr := fetch(ctx, a.Base().Key)
		Expect(userErr).To(HaveOccurred())
		Expect(rackErr).To(HaveOccurred())
		Expect(rackErr.Error()).To(Equal(userErr.Error()))
		Expect(rackErr).To(BeAssignableToTypeOf(apiarc.CompileError{}))
	})

	It("Should give a rack the program and calculations that a user's fetch of a "+
		"text automation gives", func(ctx SpecContext) {
		a := create(ctx, textBase("bk3_same_text", "bk3_in * 2.0 -> bk3_out"))
		deploy(ctx, a)
		user, rack, userErr, rackErr := fetch(ctx, a.Base().Key)
		Expect(userErr).ToNot(HaveOccurred())
		Expect(rackErr).ToNot(HaveOccurred())
		u, ok := user.Arcs[0].Variant.(arc.AutomationArc)
		Expect(ok).To(BeTrue())
		r, ok := rack.Arcs[0].Variant.(arc.AutomationArc)
		Expect(ok).To(BeTrue())
		Expect(r.Program).ToNot(BeNil())
		Expect(
			json.Marshal(*r.Program),
		).To(Equal(MustSucceed(json.Marshal(*u.Program))))
		Expect(json.Marshal(r.Calculations)).
			To(Equal(MustSucceed(json.Marshal(u.Calculations))))
	})

	It("Should give a rack the program that a user's fetch of a graph automation "+
		"gives", func(ctx SpecContext) {
		a := create(ctx, graphBase(
			"bk3_same_graph", "on", float64(in.Key()), float64(out.Key()),
		))
		deploy(ctx, a)
		user, rack, userErr, rackErr := fetch(ctx, a.Base().Key)
		Expect(userErr).ToNot(HaveOccurred())
		Expect(rackErr).ToNot(HaveOccurred())
		u, ok := user.Arcs[0].Variant.(arc.AutomationArc)
		Expect(ok).To(BeTrue())
		r, ok := rack.Arcs[0].Variant.(arc.AutomationArc)
		Expect(ok).To(BeTrue())
		Expect(r.Program).ToNot(BeNil())
		Expect(
			json.Marshal(*r.Program),
		).To(Equal(MustSucceed(json.Marshal(*u.Program))))
	})

	It("Should record a graph's node channels as deployed and keep them out of the "+
		"automation's lists at a rack's fetch", func(ctx SpecContext) {
		a := create(ctx, graphBase(
			"bk3_graph_lists", "on", float64(in.Key()), float64(out.Key()),
		))
		key := a.Base().Key
		tsk := deploy(ctx, a)
		MustSucceed(retrieve(SubjectCtx(ctx, testRack.ResourceID()), request(key)))
		rec := MustSucceed(arcSvc.RetrieveDrift(ctx, db, key))[key]
		Expect(rec.Task).To(Equal(tsk.Key))
		Expect([2][]channel.Key{rec.Reads, rec.Writes}).
			To(Equal([2][]channel.Key{{in.Key()}, {out.Key()}}))
		var stored arc.Arc
		Expect(arcSvc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.Base().Reads).To(BeEmpty())
		Expect(stored.Base().Writes).To(BeEmpty())
	})

	It("Should keep the deployed lists when a rack's fetch of its task's automation "+
		"fails to compile", func(ctx SpecContext) {
		a := create(ctx, textBase("bk3_keep", "bk3_in * 2.0 -> bk3_out"))
		key := a.Base().Key
		deploy(ctx, a)
		MustSucceed(retrieve(SubjectCtx(ctx, testRack.ResourceID()), request(key)))
		deployed := func() [2][]channel.Key {
			rec := MustSucceed(arcSvc.RetrieveDrift(ctx, db, key))[key]
			return [2][]channel.Key{rec.Reads, rec.Writes}
		}
		running := [2][]channel.Key{{in.Key()}, {out.Key()}}
		Expect(deployed()).To(Equal(running))
		// A rename leaves the text reading a name that no channel holds.
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Rename(ctx, in.Key(), "bk3_in_was", false)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Rename(ctx, in.Key(), "bk3_in", false)
			})).To(Succeed())
		})
		_, err := retrieve(SubjectCtx(ctx, testRack.ResourceID()), request(key))
		Expect(err).To(HaveOccurred())
		Expect(deployed()).To(Equal(running))
	})
})
