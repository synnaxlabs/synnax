// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
)

// fakePlugin is a minimal plugin.Plugin for exercising topoLevels.
type fakePlugin struct {
	name string
	deps []string
}

func (f *fakePlugin) Name() string                { return f.name }
func (f *fakePlugin) Domains() []string           { return nil }
func (f *fakePlugin) Requires() []string          { return f.deps }
func (f *fakePlugin) Check(*plugin.Request) error { return nil }
func (*fakePlugin) Generate(*plugin.Request) (*plugin.Response, error) {
	return &plugin.Response{}, nil
}

func registryWith(plugins ...*fakePlugin) *plugin.Registry {
	r := plugin.NewRegistry()
	for _, p := range plugins {
		Expect(r.Register(p)).To(Succeed())
	}
	return r
}

// names extracts the plugin names of a level for assertion.
func names(level []plugin.Plugin) []string {
	out := make([]string, len(level))
	for i, p := range level {
		out[i] = p.Name()
	}
	return out
}

var _ = Describe("topoLevels", func() {
	It("Should put plugins with no deps in the first level", func() {
		levels := topoLevels(registryWith(
			&fakePlugin{name: "a"},
			&fakePlugin{name: "b"},
			&fakePlugin{name: "c"},
		))
		Expect(levels).To(HaveLen(1))
		Expect(names(levels[0])).To(Equal([]string{"a", "b", "c"}))
	})

	It("Should sort within a level by name for determinism", func() {
		levels := topoLevels(registryWith(
			&fakePlugin{name: "c"},
			&fakePlugin{name: "a"},
			&fakePlugin{name: "b"},
		))
		Expect(names(levels[0])).To(Equal([]string{"a", "b", "c"}))
	})

	It("Should defer a plugin until its dep is placed", func() {
		levels := topoLevels(registryWith(
			&fakePlugin{name: "child", deps: []string{"parent"}},
			&fakePlugin{name: "parent"},
		))
		Expect(levels).To(HaveLen(2))
		Expect(names(levels[0])).To(Equal([]string{"parent"}))
		Expect(names(levels[1])).To(Equal([]string{"child"}))
	})

	It("Should batch siblings sharing a parent into one level", func() {
		levels := topoLevels(registryWith(
			&fakePlugin{name: "left", deps: []string{"root"}},
			&fakePlugin{name: "right", deps: []string{"root"}},
			&fakePlugin{name: "root"},
		))
		Expect(levels).To(HaveLen(2))
		Expect(names(levels[0])).To(Equal([]string{"root"}))
		Expect(names(levels[1])).To(Equal([]string{"left", "right"}))
	})

	It("Should layer transitive dependencies", func() {
		levels := topoLevels(registryWith(
			&fakePlugin{name: "leaf", deps: []string{"mid"}},
			&fakePlugin{name: "mid", deps: []string{"root"}},
			&fakePlugin{name: "root"},
		))
		Expect(levels).To(HaveLen(3))
		Expect(names(levels[0])).To(Equal([]string{"root"}))
		Expect(names(levels[1])).To(Equal([]string{"mid"}))
		Expect(names(levels[2])).To(Equal([]string{"leaf"}))
	})

	It("Should ignore Requires() entries that name unknown plugins", func() {
		// An unknown dep must not stall the topology — the regular
		// Check() error path is what surfaces the misconfiguration.
		levels := topoLevels(registryWith(
			&fakePlugin{name: "a", deps: []string{"missing"}},
		))
		Expect(levels).To(HaveLen(1))
		Expect(names(levels[0])).To(Equal([]string{"a"}))
	})

	It("Should fall through cyclic deps to a single final level", func() {
		levels := topoLevels(registryWith(
			&fakePlugin{name: "a", deps: []string{"b"}},
			&fakePlugin{name: "b", deps: []string{"a"}},
		))
		Expect(levels).To(HaveLen(1))
		Expect(names(levels[0])).To(ConsistOf("a", "b"))
	})
})
