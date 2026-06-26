// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pluralize_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/pluralize"
)

func TestPluralize(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pluralize Suite")
}

var _ = Describe("Pluralize", func() {
	Context("regular nouns", func() {
		It("Should add s to regular words", func() {
			Expect(pluralize.String("Channel")).To(Equal("Channels"))
			Expect(pluralize.String("Rack")).To(Equal("Racks"))
			Expect(pluralize.String("Label")).To(Equal("Labels"))
			Expect(pluralize.String("Device")).To(Equal("Devices"))
			Expect(pluralize.String("Program")).To(Equal("Programs"))
			Expect(pluralize.String("Graph")).To(Equal("Graphs"))
			Expect(pluralize.String("Node")).To(Equal("Nodes"))
			Expect(pluralize.String("Edge")).To(Equal("Edges"))
			Expect(pluralize.String("Color")).To(Equal("Colors"))
			Expect(pluralize.String("Workspace")).To(Equal("Workspaces"))
			Expect(pluralize.String("User")).To(Equal("Users"))
			Expect(pluralize.String("Log")).To(Equal("Logs"))
			Expect(pluralize.String("Table")).To(Equal("Tables"))
			Expect(pluralize.String("Task")).To(Equal("Tasks"))
			Expect(pluralize.String("State")).To(Equal("States"))
			Expect(pluralize.String("Subject")).To(Equal("Subjects"))
			Expect(pluralize.String("Command")).To(Equal("Commands"))
			Expect(pluralize.String("Pair")).To(Equal("Pairs"))
			Expect(pluralize.String("Param")).To(Equal("Params"))
			Expect(pluralize.String("Schematic")).To(Equal("Schematics"))
			Expect(pluralize.String("LinePlot")).To(Equal("LinePlots"))
			Expect(pluralize.String("Range")).To(Equal("Ranges"))
			Expect(pluralize.String("Group")).To(Equal("Groups"))
			Expect(pluralize.String("Arc")).To(Equal("Arcs"))
			Expect(pluralize.String("Handle")).To(Equal("Handles"))
			Expect(pluralize.String("TimeRange")).To(Equal("TimeRanges"))
			Expect(pluralize.String("Viewport")).To(Equal("Viewports"))
		})
	})

	Context("sibilant endings (s, x, z, ch, sh)", func() {
		It("Should add es to words ending in s", func() {
			Expect(pluralize.String("Status")).To(Equal("Statuses"))
			Expect(pluralize.String("Alias")).To(Equal("Aliases"))
			Expect(pluralize.String("bus")).To(Equal("buses"))
		})
		It("Should add es to words ending in x", func() {
			Expect(pluralize.String("box")).To(Equal("boxes"))
			Expect(pluralize.String("Index")).To(Equal("Indices"))
		})
		It("Should add es to words ending in z", func() {
			Expect(pluralize.String("quiz")).To(Equal("quizzes"))
		})
		It("Should add es to words ending in ch", func() {
			Expect(pluralize.String("catch")).To(Equal("catches"))
			Expect(pluralize.String("batch")).To(Equal("batches"))
		})
		It("Should add es to words ending in sh", func() {
			Expect(pluralize.String("crash")).To(Equal("crashes"))
			Expect(pluralize.String("flash")).To(Equal("flashes"))
		})
	})

	Context("words ending in y", func() {
		It("Should change y to ies when preceded by a consonant", func() {
			Expect(pluralize.String("Entry")).To(Equal("Entries"))
			Expect(pluralize.String("Authority")).To(Equal("Authorities"))
			Expect(pluralize.String("Body")).To(Equal("Bodies"))
			Expect(pluralize.String("Factory")).To(Equal("Factories"))
			Expect(pluralize.String("query")).To(Equal("queries"))
			Expect(pluralize.String("Category")).To(Equal("Categories"))
		})
		It("Should add s when y is preceded by a vowel", func() {
			Expect(pluralize.String("Key")).To(Equal("Keys"))
			Expect(pluralize.String("Day")).To(Equal("Days"))
			Expect(pluralize.String("XY")).To(Equal("XYs"))
			Expect(pluralize.String("Array")).To(Equal("Arrays"))
		})
		It("Should add s when a trailing acronym ends in Y", func() {
			Expect(pluralize.String("StickyXY")).To(Equal("StickyXYs"))
			Expect(pluralize.String("ClientXY")).To(Equal("ClientXYs"))
			Expect(pluralize.String("MyXY")).To(Equal("MyXYs"))
		})
	})

	Context("already plural words", func() {
		It("Should not double-pluralize words ending in es", func() {
			Expect(pluralize.String("Statuses")).To(Equal("Statuses"))
			Expect(pluralize.String("Aliases")).To(Equal("Aliases"))
			Expect(pluralize.String("Indexes")).To(Equal("Indexes"))
		})
		It("Should not double-pluralize words ending in s that are already plural", func() {
			Expect(pluralize.String("Channels")).To(Equal("Channels"))
			Expect(pluralize.String("Dimensions")).To(Equal("Dimensions"))
			Expect(pluralize.String("Authorities")).To(Equal("Authorities"))
			Expect(pluralize.String("Properties")).To(Equal("Properties"))
			Expect(pluralize.String("Sequences")).To(Equal("Sequences"))
			Expect(pluralize.String("Functions")).To(Equal("Functions"))
			Expect(pluralize.String("Stages")).To(Equal("Stages"))
			Expect(pluralize.String("Nodes")).To(Equal("Nodes"))
			Expect(pluralize.String("Edges")).To(Equal("Edges"))
		})
		It("Should not double-pluralize compound already-plural names", func() {
			Expect(pluralize.String("StatusDetails")).To(Equal("StatusDetails"))
			Expect(pluralize.String("FunctionProperties")).To(Equal("FunctionProperties"))
		})
	})

	Context("words ending in f or fe", func() {
		It("Should change f to ves", func() {
			Expect(pluralize.String("leaf")).To(Equal("leaves"))
			Expect(pluralize.String("shelf")).To(Equal("shelves"))
			Expect(pluralize.String("half")).To(Equal("halves"))
		})
		It("Should change fe to ves", func() {
			Expect(pluralize.String("knife")).To(Equal("knives"))
			Expect(pluralize.String("life")).To(Equal("lives"))
		})
	})

	Context("words ending in o", func() {
		It("Should add es to common words ending in o", func() {
			Expect(pluralize.String("hero")).To(Equal("heroes"))
			Expect(pluralize.String("potato")).To(Equal("potatoes"))
		})
	})

	Context("irregular plurals", func() {
		It("Should handle common irregular forms", func() {
			Expect(pluralize.String("child")).To(Equal("children"))
			Expect(pluralize.String("person")).To(Equal("people"))
			Expect(pluralize.String("mouse")).To(Equal("mice"))
			Expect(pluralize.String("datum")).To(Equal("data"))
			Expect(pluralize.String("criterion")).To(Equal("criteria"))
			Expect(pluralize.String("index")).To(Equal("indices"))
		})
	})

	Context("uncountable words", func() {
		It("Should return the same word for uncountable nouns", func() {
			Expect(pluralize.String("sheep")).To(Equal("sheep"))
			Expect(pluralize.String("fish")).To(Equal("fish"))
			Expect(pluralize.String("series")).To(Equal("series"))
			Expect(pluralize.String("species")).To(Equal("species"))
			Expect(pluralize.String("data")).To(Equal("data"))
			Expect(pluralize.String("metadata")).To(Equal("metadata"))
			Expect(pluralize.String("info")).To(Equal("info"))
			Expect(pluralize.String("software")).To(Equal("software"))
			Expect(pluralize.String("hardware")).To(Equal("hardware"))
			Expect(pluralize.String("firmware")).To(Equal("firmware"))
		})
	})

	Context("edge cases", func() {
		It("Should handle empty string", func() {
			Expect(pluralize.String("")).To(Equal(""))
		})
		It("Should handle single character", func() {
			Expect(pluralize.String("s")).To(Equal("ses"))
			Expect(pluralize.String("a")).To(Equal("as"))
		})
		It("Should preserve case of the original word", func() {
			Expect(pluralize.String("status")).To(Equal("statuses"))
			Expect(pluralize.String("Status")).To(Equal("Statuses"))
			Expect(pluralize.String("STATUS")).To(Equal("STATUSES"))
		})
	})
})
