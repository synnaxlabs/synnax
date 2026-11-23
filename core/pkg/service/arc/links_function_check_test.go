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
	"encoding/json/v2"
	"maps"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel links of a function name check", func() {
	var (
		key  channel.Key
		path string
	)
	BeforeEach(func(ctx SpecContext) {
		suffix := uuid.New().String()[:8]
		folder, name := "brk_links_"+suffix, "valve_auth_"+suffix
		path = folder + "." + name
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			g := group.Group{Name: folder, Scope: resource.TypeChannel}
			if err := groupSvc.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			ch := channel.Channel{
				Name:     name,
				DataType: telem.Uint8T,
				Virtual:  true,
				Group:    new(g.Key),
			}
			if err := channelSvc.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			key = ch.Key()
			return nil
		})).To(Succeed())
	})

	// createNamedChannel creates a root channel named name. The returned function
	// deletes it, and runs when the spec ends if the spec did not run it.
	createNamedChannel := func(ctx SpecContext, name string) func(SpecContext) {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		deleted := false
		remove := func(ctx SpecContext) {
			if deleted {
				return
			}
			deleted = true
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
		}
		DeferCleanup(remove)
		return remove
	}

	// createArc creates a graph Arc whose node "n" calls fn with the channel key.
	createArc := func(ctx SpecContext, fn string, cfg msgpack.EncodedJSON) arc.Arc {
		GinkgoHelper()
		inputs := msgpack.EncodedJSON{"type": fn, "channel": key}
		maps.Copy(inputs, cfg)
		a := newAutomation(arc.Base{
			Name: "brk_graph_" + uuid.New().String()[:8],
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes:  graph.Nodes{{Key: "n"}},
				Inputs: map[string]msgpack.EncodedJSON{"n": inputs},
			}}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		return a
	}

	exportFile := func(ctx SpecContext, a arc.Arc) []byte {
		GinkgoHelper()
		env := MustSucceed(svc.Export(ctx, db, arc.ResourceID(a.Base().Key)))
		return MustSucceed(json.Marshal(env.WithEncodeOptions(
			channelSvc.LinkWriter(ctx, db, func(resource.ID) bool { return true }),
		)))
	}

	nodeInputs := func(file []byte) map[string]any {
		GinkgoHelper()
		var body struct {
			Source struct {
				Graph struct {
					Inputs map[string]map[string]any `json:"inputs"`
				} `json:"graph"`
			} `json:"source"`
		}
		Expect(json.Unmarshal(file, &body)).To(Succeed())
		return body.Source.Graph.Inputs["n"]
	}

	importFile := func(ctx SpecContext, file []byte) arc.Arc {
		GinkgoHelper()
		var env imex.Envelope
		Expect(imex.Codec.Decode(file, &env)).To(Succeed())
		env.Name = "brk_imported_" + uuid.New().String()[:8]
		links := channelSvc.NewLinkReader(
			ctx,
			db,
			func(resource.ID) bool { return true },
		)
		write := MustSucceed(svc.Decode(ctx, env.WithDecodeOptions(links.Options())))
		Expect(links.Err()).ToNot(HaveOccurred())
		var id resource.ID
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = write(ctx, tx, imex.ImportOptions{})
			return err
		})).To(Succeed())
		k := MustSucceed(uuid.Parse(id.Key))
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, k)
			})).To(Succeed())
		})
		var res arc.Arc
		Expect(svc.NewRetrieve().Where(arc.MatchKeys(k)).Entry(&res).Exec(ctx, db)).
			To(Succeed())
		return res
	}

	authority := msgpack.EncodedJSON{"value": 200}

	It("Should store a key for control.set_authority when the importing Core holds "+
		"a channel named control", func(ctx SpecContext) {
		file := exportFile(ctx, createArc(ctx, "control.set_authority", authority))
		Expect(nodeInputs(file)).To(HaveKeyWithValue("channel", path))
		createNamedChannel(ctx, "control")
		res := importFile(ctx, file)
		Expect(svc.CompileProgram(ctx, res.Base().Key)).Error().ToNot(HaveOccurred())
		Expect(graphOf(res.Base()).Inputs["n"]["channel"]).
			To(BeNumerically("==", key))
	})

	It("Should compile an imported on node after the channel named on is gone",
		func(ctx SpecContext) {
			file := exportFile(ctx, createArc(ctx, "on", nil))
			Expect(nodeInputs(file)).To(HaveKeyWithValue("channel", path))
			removeOn := createNamedChannel(ctx, "on")
			res := importFile(ctx, file)
			removeOn(ctx)
			Expect(svc.CompileProgram(ctx, res.Base().Key)).Error().
				ToNot(HaveOccurred())
		})
})
