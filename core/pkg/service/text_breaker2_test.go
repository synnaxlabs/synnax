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
	"fmt"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Folder text upgrade breaker", func() {
	It("Should report a read kept beside a captured read of another channel with "+
		"its own reason",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			stand := folder("Stand A")
			// A v0.62 Core bound "pt" to the user's channel, the higher key, which the
			// built-in of the name in its folder takes from it. The top-level folder
			// takes "Metrics" from the channel.
			stored := storeV062(
				ctx,
				n,
				[]string{"pt", "pt", groupv1.MetricsFolder, "doubled"},
				func(cs []channelv1.Channel) {
					cs[0].Internal, cs[0].Group = true, &stand.Key
					cs[1].Group = &stand.Key
					cs[3].Expression = "Metrics_2 f32 := 2.0\n" +
						"return Metrics * Metrics_2 + pt"
				},
				stand,
				folder(groupv1.MetricsFolder),
			)
			MustOpen(svctestutil.OpenLayer(ctx, n))
			names := namesOf(ctx, n, stored)
			Expect(names[stored[1].Key()]).To(Equal("pt_2"))
			Expect(names[stored[2].Key()]).To(Equal("Metrics_2"))
			Expect(expressionOf(ctx, n, stored[3].Key())).
				To(Equal(stored[3].Expression))
			kept := keptStatuses(ctx, n)
			descriptions := make([]string, 0, len(kept))
			for _, s := range kept {
				descriptions = append(descriptions, s.Description)
			}
			Expect(descriptions).To(ContainElement(
				`The upgrade renamed channel "pt" to "pt_2", but the text of ` +
					`calculated channel "doubled" still reads the old name at line ` +
					`2, column 30, because the upgrade keeps a whole text when a ` +
					`symbol that it declares holds the name of one of its reads. ` +
					`Edit the text so that it reads the channel.`,
			))
		})

	It("Should write the new name of a channel under a folder that Arc cannot read "+
		"bare", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		stand, plant := folder("Stand A"), folder("plant")
		plant.Parent = &resource.ID{Type: resource.TypeGroup, Key: stand.Key.String()}
		stored := storeV062(
			ctx,
			n,
			[]string{"pt", "pt", "doubled"},
			func(cs []channelv1.Channel) {
				cs[0].Internal, cs[0].Group = true, &plant.Key
				cs[1].Group = &plant.Key
				cs[2].Expression = "return pt * 2"
			},
			stand,
			plant,
		)
		arcs := storeArcs(ctx, n, []string{"x := pt"}, []string{"{ return pt }"})
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(namesOf(ctx, n, stored)[stored[1].Key()]).To(Equal("pt_2"))
		Expect(expressionOf(ctx, n, stored[2].Key())).To(Equal("return pt_2 * 2"))
		Expect(sourceOf(ctx, n, arcs[0])).To(Equal("x := pt_2"))
		Expect(sourceOf(ctx, n, arcs[1])).To(Equal("{ return pt_2 }"))
		Expect(keptStatuses(ctx, n)).To(BeEmpty())
	})

	// A v0.62 Core moved a folder with no check of its own chain, so a store can hold a
	// folder whose chain comes back to it.
	It("Should open a v0.62 store that holds a channel in a folder whose chain comes "+
		"back to itself", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		plant, rig := folder("plant"), folder("rig")
		plant.Parent = &resource.ID{Type: resource.TypeGroup, Key: rig.Key.String()}
		rig.Parent = &resource.ID{Type: resource.TypeGroup, Key: plant.Key.String()}
		storeV062(
			ctx,
			n,
			[]string{"pt"},
			func(cs []channelv1.Channel) { cs[0].Group = &plant.Key },
			plant,
			rig,
		)
		done := make(chan error, 1)
		go func() {
			l, err := svctestutil.OpenLayer(ctx, n)
			if err == nil {
				err = l.Close()
			}
			done <- err
		}()
		select {
		case err := <-done:
			Expect(err).ToNot(HaveOccurred())
		case <-time.After(5 * time.Second):
			buf := make([]byte, 1<<16)
			Fail(fmt.Sprintf(
				"the layer did not open in 5s:\n%s",
				buf[:runtime.Stack(buf, true)],
			))
		}
	})
})
