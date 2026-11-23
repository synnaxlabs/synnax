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

var _ = Describe("Channel links of module members check", func() {
	var (
		key  channel.Key
		path string
	)
	// createChannel creates a channel named name in a new folder, deletes the channel
	// after the spec, and returns its key and path.
	createChannel := func(ctx SpecContext, name string) (channel.Key, string) {
		GinkgoHelper()
		g := group.Group{
			Name:  "arc_module_" + uuid.New().String()[:8],
			Scope: resource.TypeChannel,
		}
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := groupSvc.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			ch.Group = new(g.Key)
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
		})
		return ch.Key(), g.Name + "." + ch.Name
	}
	BeforeEach(func(ctx SpecContext) {
		key, path = createChannel(ctx, "valve_cmd_"+uuid.New().String()[:8])
	})

	// exportAuthority exports a graph Arc whose one node sets control authority on key,
	// and returns the file and the inputs of that node in it.
	exportAuthority := func(ctx SpecContext) ([]byte, map[string]any) {
		GinkgoHelper()
		a := newAutomation(arc.Base{
			Name: "authority_" + uuid.New().String()[:8],
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "auth"}},
				Inputs: map[string]msgpack.EncodedJSON{"auth": {
					"type":    "control.set_authority",
					"channel": key,
					"value":   255,
				}},
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
		env := MustSucceed(svc.Export(ctx, db, arc.ResourceID(a.Base().Key)))
		file := MustSucceed(json.Marshal(env.WithEncodeOptions(
			channelSvc.LinkWriter(ctx, db, func(resource.ID) bool { return true }),
		)))
		var body struct {
			Source struct {
				Graph struct {
					Inputs map[string]map[string]any `json:"inputs"`
				} `json:"graph"`
			} `json:"source"`
		}
		Expect(json.Unmarshal(file, &body)).To(Succeed())
		return file, body.Source.Graph.Inputs["auth"]
	}

	// importAuthority imports file as the channel link options of an import do, and
	// returns the stored inputs of its node.
	importAuthority := func(ctx SpecContext, file []byte) msgpack.EncodedJSON {
		GinkgoHelper()
		var env imex.Envelope
		Expect(imex.Codec.Decode(file, &env)).To(Succeed())
		env.Name = "imported_" + uuid.New().String()[:8]
		links := channelSvc.NewLinkReader(ctx, db)
		write := MustSucceed(svc.Decode(ctx, env.WithDecodeOptions(links.Options())))
		var id resource.ID
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = write(ctx, tx, imex.ImportOptions{})
			return err
		})).To(Succeed())
		Expect(links.Err()).To(Succeed())
		k := MustSucceed(uuid.Parse(id.Key))
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, k)
			})).To(Succeed())
		})
		var res arc.Arc
		Expect(svc.NewRetrieve().Where(arc.MatchKeys(k)).Entry(&res).Exec(ctx, db)).
			To(Succeed())
		src, ok := res.Base().Source.Variant.(arc.GraphSource)
		Expect(ok).To(BeTrue())
		return src.Graph.Inputs["auth"]
	}

	It("Should write the channel input of a module member as its path", func(
		ctx SpecContext,
	) {
		file, inputs := exportAuthority(ctx)
		Expect(inputs).To(HaveKeyWithValue("channel", path))
		Expect(importAuthority(ctx, file)).
			To(HaveKeyWithValue("channel", BeNumerically("==", key)))
	})

	It("Should write the path when a channel has the name of the module", func(
		ctx SpecContext,
	) {
		createChannel(ctx, "control")
		_, inputs := exportAuthority(ctx)
		Expect(inputs).To(HaveKeyWithValue("channel", path))
	})

	It("Should bind the path when a channel has the name of the module", func(
		ctx SpecContext,
	) {
		file, inputs := exportAuthority(ctx)
		Expect(inputs).To(HaveKeyWithValue("channel", path))
		createChannel(ctx, "control")
		Expect(importAuthority(ctx, file)).
			To(HaveKeyWithValue("channel", BeNumerically("==", key)))
	})
})
