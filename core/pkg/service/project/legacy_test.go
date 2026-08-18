// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"encoding/json"
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/zip"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// legacyLayoutFile builds a LAYOUT.json holding one layout record per entry.
func legacyLayoutFile(layouts map[string]map[string]any) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{"layouts": layouts}))
}

// legacyLogState is a frozen legacy Console log state: typeless, recognized by its
// channels array.
func legacyLogState(extra map[string]any) []byte {
	GinkgoHelper()
	body := map[string]any{
		"version":       "0.0.0",
		"channels":      []any{4, 5},
		"remoteCreated": false,
	}
	maps.Copy(body, extra)
	return MustSucceed(json.Marshal(body))
}

var _ = Describe("Legacy import", func() {
	importLegacy := func(ctx SpecContext, files zip.Files) project.Project {
		GinkgoHelper()
		return MustSucceed(svc.Import(ctx, tx, files, "Legacy Project.zip"))
	}
	logLayout := map[string]map[string]any{
		"k1": {"key": "k1", "type": "log", "name": "Metrics"},
	}

	It("Should name the project after the file name", func(ctx SpecContext) {
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(nil),
		})
		Expect(proj.Name).To(Equal("Legacy Project"))
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(BeEmpty())
	})

	DescribeTable("Should locate a layout's component file",
		func(ctx SpecContext, fileName string, extra map[string]any, logName string) {
			proj := importLegacy(ctx, zip.Files{
				"LAYOUT.json": legacyLayoutFile(logLayout),
				fileName:      legacyLogState(extra),
			})
			children := childrenOf(ctx, project.OntologyID(proj.Key))
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID.Type).To(Equal(ontology.ResourceTypeLog))
			Expect(children[0].Name).To(Equal(logName))
		},
		Entry("named after the layout", "Metrics.json", nil, "Metrics"),
		Entry("named after the layout key", "k1.json", nil, "k1"),
		Entry("matched by its body key", "State.json",
			map[string]any{"key": "k1"}, "State"),
		Entry("matched by its body name", "State.json",
			map[string]any{"name": "Metrics"}, "Metrics"),
	)

	It("Should skip layouts whose type is not a frozen legacy type", func(
		ctx SpecContext,
	) {
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
				"m1": {"key": "m1", "type": "mosaic", "name": "Main"},
			}),
		})
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(BeEmpty())
	})

	It("Should import legacy task files through the leaf machinery", func(
		ctx SpecContext,
	) {
		t := createTask(ctx, "Sequence")
		env := MustSucceed(imexSvc.Export(ctx, task.OntologyID(t.Key)))
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
				"t1": {"key": "t1", "type": "pagerduty_alert", "name": "Sequence"},
			}),
			"Sequence.json": MustSucceed(json.Marshal(env)),
		})
		// The recreated task parents under the rack, so the project stays empty.
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(BeEmpty())
		var tasks []task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchNames("Sequence")).
			Entries(&tasks).
			Exec(ctx, tx)).To(Succeed())
		Expect(tasks).To(HaveLen(2))
	})

	It("Should reject a layout whose component file is missing", func(
		ctx SpecContext,
	) {
		files := zip.Files{"LAYOUT.json": legacyLayoutFile(logLayout)}
		Expect(svc.Import(ctx, tx, files, "Legacy.zip")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`data for layout "k1" not found`)),
		))
	})

	It("Should list the legacy members' types for access checks", func(
		ctx SpecContext,
	) {
		objects := MustSucceed(svc.ImportObjects(ctx, zip.Files{
			"LAYOUT.json":  legacyLayoutFile(logLayout),
			"Metrics.json": legacyLogState(nil),
		}))
		Expect(objects).To(ConsistOf(
			ontology.ID{Type: ontology.ResourceTypeProject},
			ontology.ID{Type: ontology.ResourceTypeLog},
		))
	})
})
