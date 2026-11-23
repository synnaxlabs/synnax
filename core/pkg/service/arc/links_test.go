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
	"fmt"
	"strings"
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
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Channel links", func() {
	var (
		key  channel.Key
		path string
	)
	BeforeEach(func(ctx SpecContext) {
		suffix := uuid.New().String()[:8]
		folder, name := "arc_links_"+suffix, "valve_cmd_"+suffix
		path = folder + "." + name
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			g := group.Group{Name: folder, Scope: resource.TypeChannel}
			if err := groupSvc.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			ch := channel.Channel{
				Name:     name,
				DataType: telem.Float32T,
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

	// createGraphArcWith creates a graph Arc whose source node reads input and whose
	// sink node writes the value 3 to input.
	createGraphArcWith := func(ctx SpecContext, input any) arc.Arc {
		GinkgoHelper()
		a := newAutomation(arc.Base{
			Name: "graph_" + uuid.New().String()[:8],
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "source"}, {Key: "sink"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {"type": "on", "channel": input},
					"sink":   {"type": "write", "channel": input, "value": 3},
				},
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

	// createGraphArc creates a graph Arc whose nodes read and write key.
	createGraphArc := func(ctx SpecContext) arc.Arc {
		GinkgoHelper()
		return createGraphArcWith(ctx, key)
	}

	// inputsOf returns the inputs of each node of the graph in a file's body.
	inputsOf := func(file []byte) map[string]map[string]any {
		GinkgoHelper()
		var body struct {
			Source struct {
				Graph struct {
					Inputs map[string]map[string]any `json:"inputs"`
				} `json:"graph"`
			} `json:"source"`
		}
		Expect(json.Unmarshal(file, &body)).To(Succeed())
		return body.Source.Graph.Inputs
	}

	// importFile decodes file with the channel link options of an import, writes the
	// Arc, and returns it with the error of the link options.
	importFile := func(ctx SpecContext, file []byte) (arc.Arc, error) {
		GinkgoHelper()
		var env imex.Envelope
		Expect(imex.Codec.Decode(file, &env)).To(Succeed())
		env.Name = "imported_" + uuid.New().String()[:8]
		links := channelSvc.NewLinkReader(
			ctx,
			db,
			func(resource.ID) bool { return true },
		)
		write := MustSucceed(svc.Decode(ctx, env.WithDecodeOptions(links.Options())))
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
		return res, links.Err()
	}

	exportFile := func(ctx SpecContext, a arc.Arc) []byte {
		GinkgoHelper()
		env := MustSucceed(svc.Export(ctx, db, arc.ResourceID(a.Base().Key)))
		return MustSucceed(json.Marshal(env.WithEncodeOptions(
			channelSvc.LinkWriter(ctx, db, func(resource.ID) bool { return true }),
		)))
	}

	It("Should write each channel input as its path and leave other inputs", func(
		ctx SpecContext,
	) {
		inputs := inputsOf(exportFile(ctx, createGraphArc(ctx)))
		Expect(inputs["source"]).To(HaveKeyWithValue("channel", path))
		Expect(inputs["sink"]).To(HaveKeyWithValue("channel", path))
		Expect(inputs["sink"]).To(HaveKeyWithValue("value", BeNumerically("==", 3)))
	})

	It("Should export a null channel input as no channel", func(ctx SpecContext) {
		inputs := inputsOf(exportFile(ctx, createGraphArcWith(ctx, nil)))
		Expect(inputs["source"]).To(HaveKeyWithValue("channel", BeNumerically("==", 0)))
	})

	It("Should fail the export of a channel input that holds text", func(
		ctx SpecContext,
	) {
		a := createGraphArcWith(ctx, "pump.speed")
		Expect(svc.Export(ctx, db, arc.ResourceID(a.Base().Key))).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"Arc %s: input channel of node source: holds text, not a channel key",
				a.Base().Name,
			)),
		))
	})

	It("Should fail the export of a channel input that holds an invalid number", func(
		ctx SpecContext,
	) {
		a := createGraphArcWith(ctx, -1)
		Expect(svc.Export(ctx, db, arc.ResourceID(a.Base().Key))).Error().
			To(MatchError(ContainSubstring("Arc %s: input channel of node source", a.Base().Name)))
	})

	It("Should bind a channel input named by path on import", func(ctx SpecContext) {
		res := MustSucceed(importFile(ctx, exportFile(ctx, createGraphArc(ctx))))
		g := graphOf(res.Base())
		Expect(g.Inputs["source"]["channel"]).To(BeNumerically("==", key))
		Expect(g.Inputs["sink"]["channel"]).To(BeNumerically("==", key))
		Expect(g.Inputs["sink"]["value"]).To(BeNumerically("==", 3))
	})

	It("Should bind a channel input that holds a raw key", func(ctx SpecContext) {
		a := createGraphArc(ctx)
		env := MustSucceed(svc.Export(ctx, db, arc.ResourceID(a.Base().Key)))
		res := MustSucceed(importFile(ctx, MustSucceed(json.Marshal(env))))
		Expect(graphOf(res.Base()).Inputs["source"]["channel"]).
			To(BeNumerically("==", key))
	})

	It("Should bind a raw key and a path in a graph file that a Console wrote", func(
		ctx SpecContext,
	) {
		file := fmt.Sprintf(`{
			"key": %q,
			"type": "arc",
			"version": "0.0.0",
			"mode": "graph",
			"text": {"raw": ""},
			"graph": {
				"nodes": [
					{"key": "by_key", "position": {"x": 0, "y": 0}},
					{"key": "by_path", "position": {"x": 0, "y": 0}}
				],
				"edges": [],
				"props": {
					"by_key": {"key": "on", "channel": %d},
					"by_path": {"key": "on", "channel": %q}
				}
			}
		}`, uuid.New(), key, path)
		g := graphOf(MustSucceed(importFile(ctx, []byte(file))).Base())
		Expect(g.Inputs["by_key"]["channel"]).To(BeNumerically("==", key))
		Expect(g.Inputs["by_path"]["channel"]).To(BeNumerically("==", key))
	})

	It("Should list a channel input whose path names no channel", func(
		ctx SpecContext,
	) {
		file := strings.ReplaceAll(
			string(exportFile(ctx, createGraphArc(ctx))),
			fmt.Sprintf("%q", path),
			fmt.Sprintf("%q", path+"_gone"),
		)
		res, err := importFile(ctx, []byte(file))
		Expect(err).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("no channel is at %q", path+"_gone")),
		))
		Expect(graphOf(res.Base()).Inputs["source"]["channel"]).
			To(BeNumerically("==", 0))
	})
})
