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
	"context"
	"encoding/json"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelversions "github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupversions "github.com/synnaxlabs/synnax/pkg/service/group/versions"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	resourcev0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// markAppliedBefore records each step of chain before the step with key next as applied
// in namespace, as a v0.62 Core left it.
func markAppliedBefore(
	ctx context.Context,
	db *gorp.DB,
	namespace string,
	chain []gorp.Migration,
	next string,
) {
	GinkgoHelper()
	end := slices.IndexFunc(
		chain,
		func(m gorp.Migration) bool { return m.Key() == next },
	)
	Expect(end).To(BeNumerically(">", 0))
	keys := make([]string, end)
	for i, m := range chain[:end] {
		keys[i] = m.Key()
	}
	markApplied(ctx, db, namespace, keys...)
}

// storeV062 makes n hold a store as a v0.62 Core left it: the chain steps before the
// name passes applied, a channel v1 record for each name, and the folders. edit changes
// the records before they are stored. It returns the records.
func storeV062(
	ctx context.Context,
	n testutil.Node,
	names []string,
	edit func([]channelv1.Channel),
	folders ...groupv1.Group,
) []channelv1.Channel {
	GinkgoHelper()
	markAppliedBefore(
		ctx,
		n.DB,
		"Channel",
		channelversions.Migrations,
		"v63_channel_name_rule",
	)
	markAppliedBefore(
		ctx,
		n.DB,
		"Group",
		groupversions.Migrations,
		"v63_folder_name_rule",
	)
	alloc := make([]distchannel.Channel, len(names))
	for i, name := range names {
		alloc[i] = distchannel.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			Virtual:     true,
		}
	}
	keys := MustSucceed(n.Channel.Allocate(ctx, alloc))
	stored := make([]channelv1.Channel, len(keys))
	for i, k := range keys {
		stored[i] = channelv1.Channel{
			Name:        k.Name,
			Leaseholder: node.KeyFree,
			LocalKey:    k.LocalKey,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
	}
	edit(stored)
	Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
		Set(ctx, stored...)).To(Succeed())
	Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](n.DB).Set(ctx, folders...)).
		To(Succeed())
	return stored
}

// folder returns a top-level channel folder named name.
func folder(name string) groupv1.Group {
	return groupv1.Group{Key: uuid.New(), Name: name, Scope: resource.TypeChannel}
}

// storeArcs opens the Arc table of n once, as v0.62 did, and stores an Arc with each
// text source and one with a graph function of each body, in its root directory. It
// returns the keys of the Arcs.
func storeArcs(ctx context.Context, n testutil.Node, texts, bodies []string) []arc.Key {
	GinkgoHelper()
	Expect(MustSucceed(arc.OpenTable(ctx, arc.TableConfig{DB: n.DB})).Close()).
		To(Succeed())
	isRoot := func(_ gorp.Context, d *directory.Directory) (bool, error) {
		return d.Parent == nil, nil
	}
	var root directory.Directory
	Expect(gorp.NewRetrieve[directory.Key, directory.Directory]().
		Where(gorp.Match(isRoot)).
		Entry(&root).
		Exec(ctx, n.DB)).To(Succeed())
	arcs := make([]arc.Arc, 0, len(texts)+len(bodies))
	for _, raw := range texts {
		arcs = append(arcs, arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Key:       uuid.New(),
			Name:      "text",
			Directory: &root.Key,
			Source: arc.Source{Variant: arc.TextSource{
				Text: text.Text{Doc: text.Create(raw)},
			}},
		}}})
	}
	for _, body := range bodies {
		arcs = append(arcs, arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Key:       uuid.New(),
			Name:      "graph",
			Directory: &root.Key,
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Functions: ir.Functions{{Key: "f", Body: ir.Body{Raw: body}}},
			}}},
		}}})
	}
	Expect(gorp.WrapWriter[arc.Key, arc.Arc](n.DB).Set(ctx, arcs...)).To(Succeed())
	keys := make([]arc.Key, len(arcs))
	for i, a := range arcs {
		keys[i] = a.Base().Key
	}
	return keys
}

// sourceOf returns the text of the Arc with key, or the body of its graph function.
func sourceOf(ctx context.Context, n testutil.Node, key arc.Key) string {
	GinkgoHelper()
	var a arc.Arc
	Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
		Where(gorp.MatchKeys[arc.Key, arc.Arc](key)).
		Entry(&a).
		Exec(ctx, n.DB)).To(Succeed())
	switch src := a.Base().Source.Variant.(type) {
	case arc.TextSource:
		return src.Text.Materialize().Raw
	case arc.GraphSource:
		return src.Graph.Functions[0].Body.Raw
	}
	Fail("unexpected source")
	return ""
}

// namesOf returns the name of each stored channel by key.
func namesOf(
	ctx context.Context,
	n testutil.Node,
	stored []channelv1.Channel,
) map[channel.Key]string {
	GinkgoHelper()
	keys := make([]channel.Key, len(stored))
	for i, c := range stored {
		keys[i] = c.Key()
	}
	var chs []channel.Channel
	Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
		Where(gorp.MatchKeys[channel.Key, channel.Channel](keys...)).
		Entries(&chs).
		Exec(ctx, n.DB)).To(Succeed())
	names := make(map[channel.Key]string, len(chs))
	for _, c := range chs {
		names[c.Key()] = c.Name
	}
	return names
}

// renameStatuses returns each rename status by the name it reports.
func renameStatuses(
	ctx context.Context,
	n testutil.Node,
) map[string]status.Status[any] {
	GinkgoHelper()
	isRename := func(_ gorp.Context, s *status.Status[any]) (bool, error) {
		return s.Message == "Renamed channel" || s.Message == "Renamed folder", nil
	}
	var statuses []status.Status[any]
	Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
		Where(gorp.Match(isRename)).
		Entries(&statuses).
		Exec(ctx, n.DB)).To(Succeed())
	byName := make(map[string]status.Status[any], len(statuses))
	for _, s := range statuses {
		Expect(s.Variant).To(Equal(status.VariantWarning))
		byName[s.Name] = s
	}
	return byName
}

var _ = Describe("Name upgrade", func() {
	It("Should rename, quote, and report the names of a v0.62 store", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		rigs, ab, xy, ac := folder("Rigs"), folder("a_b"), folder("x.y"), folder("a.c")
		stored := storeV062(ctx, n, []string{
			"pt-01",
			"stand.pt",
			"pressure",
			"pressure",
			"Rigs",
			"a.b",
			"doubled",
			"x.y",
			"a_b_2",
			"tripled",
			"a_c",
		}, func(cs []channelv1.Channel) {
			cs[6].Expression = "return pt-01 * 2"
			// A channel in another folder holds the first suffix of a moved channel.
			cs[8].Group = &rigs.Key
			// The lower key keeps the name, so the text keeps naming it.
			cs[9].Expression = "return pressure * 3"
		}, rigs, ab, xy, ac)
		arcs := storeArcs(
			ctx,
			n,
			[]string{"x := pt-01 * 2 + pt-02 // pt-01"},
			[]string{"{ return pt-01 }"},
		)

		l := MustOpen(svctestutil.OpenLayer(ctx, n))

		Expect(namesOf(ctx, n, stored)).To(Equal(map[channel.Key]string{
			stored[0].Key():  "pt-01",
			stored[1].Key():  "stand_pt",
			stored[2].Key():  "pressure",
			stored[3].Key():  "pressure_2",
			stored[4].Key():  "Rigs",
			stored[5].Key():  "a_b_3",
			stored[6].Key():  "doubled",
			stored[7].Key():  "x_y",
			stored[8].Key():  "a_b_2",
			stored[9].Key():  "tripled",
			stored[10].Key(): "a_c",
		}))
		var folders []group.Group
		Expect(gorp.NewRetrieve[group.Key, group.Group]().
			Where(gorp.MatchKeys[group.Key, group.Group](
				rigs.Key,
				ab.Key,
				xy.Key,
				ac.Key,
			)).
			Entries(&folders).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(folders).To(ConsistOf(
			HaveField("Name", "Rigs_2"),
			HaveField("Name", "a_b"),
			HaveField("Name", "x_y_2"),
			HaveField("Name", "a_c_2"),
		))
		doubled := calculationOf(ctx, n, l.Calculation, stored[6].Key())
		Expect(doubled.Expression).To(Equal("return 'pt-01' * 2"))
		Expect(doubled.Reads).To(Equal([]channel.Key{stored[0].Key()}))
		Expect(calculationOf(ctx, n, l.Calculation, stored[9].Key()).Expression).
			To(Equal("return pressure * 3"))
		Expect(sourceOf(ctx, n, arcs[0])).
			To(Equal("x := 'pt-01' * 2 + pt-02 // pt-01"))
		Expect(sourceOf(ctx, n, arcs[1])).To(Equal("{ return 'pt-01' }"))
		statuses := renameStatuses(ctx, n)
		reported := make(map[string]string, len(statuses))
		for name, s := range statuses {
			reported[name] = s.Message
		}
		Expect(reported).To(Equal(map[string]string{
			"stand_pt":   "Renamed channel",
			"pressure_2": "Renamed channel",
			"a_b_3":      "Renamed channel",
			"x_y":        "Renamed channel",
			"Rigs_2":     "Renamed folder",
			"x_y_2":      "Renamed folder",
			"a_c_2":      "Renamed folder",
		}))
		Expect(statuses["a_b_3"].Description).To(ContainSubstring(`"a.b" to "a_b_3"`))
		Expect(MustSucceed(resourcev0.StagedRenames(n.DB))).To(BeEmpty())
	})

	It("Should move a top-level channel off the name of the Metrics folder and write "+
		"its new name into text", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		metrics := folder(groupv1.MetricsFolder)
		stored := storeV062(ctx, n, []string{"Metrics", "doubled"}, func(
			cs []channelv1.Channel,
		) {
			cs[1].Expression = "return Metrics * 2"
		}, metrics)
		arcs := storeArcs(ctx, n, []string{"x := Metrics * 2"}, nil)
		check := func(l *service.Layer) {
			GinkgoHelper()
			Expect(namesOf(ctx, n, stored)[stored[0].Key()]).To(Equal("Metrics_2"))
			var f group.Group
			Expect(gorp.NewRetrieve[group.Key, group.Group]().
				Where(gorp.MatchKeys[group.Key, group.Group](metrics.Key)).
				Entry(&f).
				Exec(ctx, n.DB)).To(Succeed())
			Expect(f.Name).To(Equal(groupv1.MetricsFolder))
			Expect(calculationOf(ctx, n, l.Calculation, stored[1].Key()).Expression).
				To(Equal("return 'Metrics_2' * 2"))
			Expect(sourceOf(ctx, n, arcs[0])).To(Equal("x := 'Metrics_2' * 2"))
			statuses := renameStatuses(ctx, n)
			Expect(statuses).To(HaveLen(1))
			Expect(statuses["Metrics_2"].Description).
				To(ContainSubstring(`"Metrics" to "Metrics_2"`))
		}
		first := MustSucceed(svctestutil.OpenLayer(ctx, n))
		firstOpen := true
		DeferCleanup(func() {
			if firstOpen {
				Expect(first.Close()).To(Succeed())
			}
		})
		check(first)
		firstOpen = false
		Expect(first.Close()).To(Succeed())
		check(MustOpen(svctestutil.OpenLayer(ctx, n)))
	})

	It("Should leave what each Arc shows as changed since its deploy", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		first := MustSucceed(svctestutil.OpenLayer(ctx, n))
		firstOpen := true
		DeferCleanup(func() {
			if firstOpen {
				Expect(first.Close()).To(Succeed())
			}
		})
		pt := channel.Channel{
			Name:        "pt-01",
			DataType:    telem.Float32T,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
		bench := rack.Rack{Name: "bench"}
		automation := func(name, raw string) arc.Arc {
			return arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
				Name: name,
				Source: arc.Source{Variant: arc.TextSource{
					Text: text.Text{Doc: text.Create(raw)},
				}},
			}}}
		}
		running := automation("running", "x := pt-01 * 2")
		edited := automation("edited", "y := pt-01 * 3")
		var runningTask, editedTask *task.Task
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			if err = first.Channel.NewWriter(tx).Create(ctx, &pt); err != nil {
				return err
			}
			if err = first.Rack.NewWriter(tx).Create(ctx, &bench); err != nil {
				return err
			}
			w := first.Arc.NewWriter(tx)
			if err = w.Create(ctx, &running); err != nil {
				return err
			}
			if err = w.Create(ctx, &edited); err != nil {
				return err
			}
			update := arc.TaskUpdate{Rack: &bench.Key}
			if runningTask, err = w.UpdateTask(
				ctx,
				running.Base().Key,
				update,
			); err != nil {
				return err
			}
			editedTask, err = w.UpdateTask(ctx, edited.Base().Key, update)
			return err
		})).To(Succeed())
		firstOpen = false
		Expect(first.Close()).To(Succeed())
		// The Driver ran each deploy, and the edited Arc then saved an edit.
		deployed := drift.Hashes{Text: "deployed"}
		Expect(gorp.WrapWriter[drift.Key, drift.Record](n.DB).Set(
			ctx,
			drift.Record{Task: runningTask.Key, Stored: deployed, Deployed: deployed},
			drift.Record{
				Task:     editedTask.Key,
				Stored:   drift.Hashes{Text: "edited"},
				Deployed: deployed,
			},
		)).To(Succeed())
		// The store is as v0.62 left it: the Arc text pass has not run.
		record := []byte("gorp.migration.Composition")
		b, closer := MustSucceed2(n.Storage.KV.Get(record))
		var applied []string
		Expect(json.Unmarshal(b, &applied)).To(Succeed())
		Expect(closer.Close()).To(Succeed())
		applied = slices.DeleteFunc(applied, func(k string) bool {
			return k == "v63_rename_arc_channels"
		})
		Expect(n.Storage.KV.Set(ctx, record, MustSucceed(json.Marshal(applied)))).
			To(Succeed())

		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(sourceOf(ctx, n, running.Base().Key)).To(Equal("x := 'pt-01' * 2"))
		// The Driver configures the running task again from the stored text.
		var current task.Task
		Expect(l.Task.NewRetrieve().
			Where(task.MatchKeys(runningTask.Key)).
			Entry(&current).
			Exec(ctx, n.DB)).To(Succeed())
		stat := &task.Status{
			Owner:   new(task.ResourceID(current.Key)),
			Name:    "running",
			Variant: status.VariantSuccess,
			Time:    telem.Now(),
			Details: task.StatusDetails{Running: true, ConfigHash: current.ConfigHash},
		}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Status.NewWriter(tx).Set(ctx, stat)
		})).To(Succeed())
		records := MustSucceed(l.Arc.RetrieveDrift(
			ctx,
			n.DB,
			running.Base().Key,
			edited.Base().Key,
		))
		Expect(arc.Changes(records[running.Base().Key])).To(BeEmpty())
		Expect(arc.Changes(records[edited.Base().Key])).To(Equal([]arc.Change{
			{Type: arc.TypeAutomation, Kind: arc.ChangeKindChanged},
		}))
	})

	It("Should rename the storage of a renamed channel", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		storeV062(ctx, n, nil, func([]channelv1.Channel) {})
		idx := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
			Name:     "stand.time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}}))[0]
		Expect(n.Storage.TS.CreateChannel(
			ctx,
			distchannel.ToStorage([]distchannel.Channel{idx})...,
		)).To(Succeed())
		Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).Set(
			ctx,
			channelv1.Channel{
				Name:        idx.Name,
				Leaseholder: idx.Leaseholder,
				LocalKey:    idx.LocalKey,
				LocalIndex:  idx.LocalKey,
				IsIndex:     true,
				DataType:    telem.TimestampT,
			},
		)).To(Succeed())
		MustOpen(svctestutil.OpenLayer(ctx, n))
		stored := MustSucceed(
			n.Storage.TS.RetrieveChannel(ctx, cesium.ChannelKey(idx.Key())),
		)
		Expect(stored.Name).To(Equal("stand_time"))
	})
})
