// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bundle_test

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
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/defaults"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// createFolderChannel creates a channel named name in a new channel folder named
// folder, and returns its key.
func createFolderChannel(ctx SpecContext, folder, name string) channel.Key {
	GinkgoHelper()
	var key channel.Key
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
	return key
}

// createLinkedLog creates a log in proj with one entry per key.
func createLinkedLog(ctx SpecContext, proj project.Key, keys ...channel.Key) {
	GinkgoHelper()
	l := defaults.Fill(log.Log{Project: &proj, Name: "Pressures"})
	for _, k := range keys {
		l.Channels = append(l.Channels, defaults.Fill(log.ChannelEntry{
			Channel:  k,
			Notation: notation.NotationStandard,
		}))
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return logSvc.NewWriter(tx).Create(ctx, &l)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return logSvc.NewWriter(tx).Delete(ctx, l.Key)
		})).To(Succeed())
	})
}

// memberOfType returns the decoded body of the one file in files of type t.
func memberOfType(files zip.Files, t string) map[string]any {
	GinkgoHelper()
	var found []map[string]any
	for _, b := range files {
		if m := decode(b); m["type"] == t {
			found = append(found, m)
		}
	}
	Expect(found).To(HaveLen(1))
	return found[0]
}

// importedLogChannels returns the channels of the entries of the one log in proj.
func importedLogChannels(ctx SpecContext, proj project.Key) []channel.Key {
	GinkgoHelper()
	id := childOfType(childrenOf(ctx, project.ResourceID(proj)), resource.TypeLog).ID
	var l log.Log
	Expect(logSvc.NewRetrieve().
		Where(log.MatchKeys(MustSucceed(uuid.Parse(id.Key)))).
		Entry(&l).
		Exec(ctx, tx)).To(Succeed())
	keys := make([]channel.Key, len(l.Channels))
	for i, e := range l.Channels {
		keys[i] = e.Channel
	}
	return keys
}

var _ = Describe("Channel links", func() {
	It("Should name each channel by its path and bind it again on import", func(
		ctx SpecContext,
	) {
		folder, name := "links_"+uuid.New().String()[:8], "pt_01"
		key := createFolderChannel(ctx, folder, name)
		src := createProject(ctx, "Links")
		createLinkedLog(ctx, src.Key, key)
		files, _ := MustSucceed2(svc.Export(ctx, db, src.Key, xjson.Codec))
		Expect(memberOfType(files, "log")["channels"]).To(ConsistOf(
			HaveKeyWithValue("channel", folder+"."+name),
		))
		tx = DeferClose(db.OpenTx())
		proj := MustSucceed(importFiles(ctx, files, "Fallback.zip"))
		Expect(importedLogChannels(ctx, proj.Key)).To(Equal([]channel.Key{key}))
	})

	It("Should list every missing path and import nothing", func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		files := zip.Files{
			"manifest.json": bundleManifest("Missing"),
			"Pressures.json": MustSucceed(json.Marshal(map[string]any{
				"version": 5,
				"type":    "log",
				"name":    "Pressures",
				"channels": []any{
					map[string]any{"channel": "plant.b"},
					map[string]any{"channel": "plant.a"},
				},
			})),
		}
		Expect(importFiles(ctx, files, "Missing.zip")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("no channel is at plant.a, plant.b")),
		))
	})

	It("Should name a graph Arc's channel inputs by path beside the other files", func(
		ctx SpecContext,
	) {
		folder, name := "links_"+uuid.New().String()[:8], "pt_02"
		key := createFolderChannel(ctx, folder, name)
		src := createProject(ctx, "Mixed")
		createLinkedLog(ctx, src.Key, key)
		a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Name: "Graph " + folder,
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "n1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"n1": {"type": "on", "channel": key},
				},
			}}},
		}}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		createPanel(ctx, "Controls", src.Key, leaf(resourceTab(a.ResourceID())))
		files, _ := MustSucceed2(svc.Export(ctx, db, src.Key, xjson.Codec))
		Expect(memberOfType(files, "log")["channels"]).To(ConsistOf(
			HaveKeyWithValue("channel", folder+"."+name),
		))
		node := HaveKeyWithValue("n1", HaveKeyWithValue("channel", folder+"."+name))
		Expect(memberOfType(files, "arc-automation")["source"]).To(
			HaveKeyWithValue("graph", HaveKeyWithValue("inputs", node)),
		)
		// The import lands beside the original, where the name is taken.
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).Apply(
				ctx,
				a.Base().Key,
				arc.NewSetAction(arc.SetPayload{
					Path:  []string{"name"},
					Value: "Source " + folder,
				}),
			)
		})).To(Succeed())
		tx = DeferClose(db.OpenTx())
		proj := MustSucceed(importFiles(ctx, files, "Fallback.zip"))
		Expect(importedLogChannels(ctx, proj.Key)).To(Equal([]channel.Key{key}))
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		panelID := childOfType(children, resource.TypePanel).ID
		arcID := firstTabResource(retrievePanel(ctx, panelID))
		var res arc.Arc
		Expect(arcSvc.NewRetrieve().
			Where(arc.MatchKeys(MustSucceed(uuid.Parse(arcID.Key)))).
			Entry(&res).
			Exec(ctx, tx)).To(Succeed())
		graphSrc, ok := res.Base().Source.Variant.(arc.GraphSource)
		Expect(ok).To(BeTrue())
		Expect(graphSrc.Graph.Inputs["n1"]["channel"]).To(BeNumerically("==", key))
	})
})
