// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
)

var _ = Describe("ImportSet", func() {
	Describe("NewImportSet", func() {
		It("Should return a gated set", func() {
			s := symbol.NewImportSet()
			Expect(s.AutoAll()).To(BeFalse())
		})
	})

	Describe("NewAutoImportSet", func() {
		It("Should return a set with AutoAll=true", func() {
			s := symbol.NewAutoImportSet()
			Expect(s.AutoAll()).To(BeTrue())
		})

		It("Should return nil from Unused regardless of unused records", func() {
			s := symbol.NewAutoImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			Expect(s.Unused()).To(BeNil())
		})
	})

	Describe("AutoAll", func() {
		It("Should be nil-safe", func() {
			var s *symbol.ImportSet
			Expect(s.AutoAll()).To(BeFalse())
		})
	})

	Describe("Lookup", func() {
		It("Should return the record bound to an alias", func() {
			s := symbol.NewImportSet()
			rec := symbol.ImportRecord{Path: "time", Alias: "t"}
			s.Add(rec)
			got, ok := s.Lookup("t")
			Expect(ok).To(BeTrue())
			Expect(got).To(Equal(rec))
		})

		It("Should report ok=false for unbound aliases", func() {
			s := symbol.NewImportSet()
			_, ok := s.Lookup("missing")
			Expect(ok).To(BeFalse())
		})

		It("Should be nil-safe", func() {
			var s *symbol.ImportSet
			_, ok := s.Lookup("anything")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Add", func() {
		It("Should insert a new record and return false", func() {
			s := symbol.NewImportSet()
			duplicate := s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			Expect(duplicate).To(BeFalse())
		})

		It("Should return true when the alias is already bound", func() {
			s := symbol.NewImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			duplicate := s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			Expect(duplicate).To(BeTrue())
		})

		It("Should be nil-safe", func() {
			var s *symbol.ImportSet
			Expect(s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})).To(BeFalse())
		})
	})

	Describe("MarkUsed", func() {
		It("Should flag the record as used", func() {
			s := symbol.NewImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			s.MarkUsed("time")
			Expect(s.Unused()).To(BeEmpty())
		})

		It("Should be a no-op for unknown aliases", func() {
			s := symbol.NewImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			s.MarkUsed("nope")
			Expect(s.Unused()).To(HaveLen(1))
		})

		It("Should be nil-safe", func() {
			var s *symbol.ImportSet
			Expect(func() { s.MarkUsed("anything") }).ToNot(Panic())
		})
	})

	Describe("All", func() {
		It("Should return every record", func() {
			s := symbol.NewImportSet()
			a := symbol.ImportRecord{Path: "time", Alias: "time"}
			b := symbol.ImportRecord{Path: "math", Alias: "math"}
			s.Add(a)
			s.Add(b)
			Expect(s.All()).To(ConsistOf(a, b))
		})

		It("Should be nil-safe", func() {
			var s *symbol.ImportSet
			Expect(s.All()).To(BeNil())
		})
	})

	Describe("Unused", func() {
		It("Should return records that were never marked used", func() {
			s := symbol.NewImportSet()
			used := symbol.ImportRecord{Path: "time", Alias: "time"}
			unused := symbol.ImportRecord{Path: "math", Alias: "math"}
			s.Add(used)
			s.Add(unused)
			s.MarkUsed("time")
			Expect(s.Unused()).To(ConsistOf(unused))
		})

		It("Should return nil for an auto-import set", func() {
			s := symbol.NewAutoImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			Expect(s.Unused()).To(BeNil())
		})

		It("Should be nil-safe", func() {
			var s *symbol.ImportSet
			Expect(s.Unused()).To(BeNil())
		})
	})

	Describe("CanonicalName", func() {
		It("Should rewrite an aliased prefix to the module path", func() {
			s := symbol.NewImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "t"})
			Expect(s.CanonicalName("t.now")).To(Equal("time.now"))
		})

		It("Should leave an unaliased qualified name unchanged", func() {
			s := symbol.NewImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "time"})
			Expect(s.CanonicalName("time.now")).To(Equal("time.now"))
		})

		It("Should leave unqualified names unchanged", func() {
			s := symbol.NewImportSet()
			s.Add(symbol.ImportRecord{Path: "time", Alias: "t"})
			Expect(s.CanonicalName("bare_name")).To(Equal("bare_name"))
		})

		It("Should leave names with unbound prefixes unchanged", func() {
			s := symbol.NewImportSet()
			Expect(s.CanonicalName("unknown.thing")).To(Equal("unknown.thing"))
		})

		It("Should be nil-safe", func() {
			var s *symbol.ImportSet
			Expect(s.CanonicalName("t.now")).To(Equal("t.now"))
		})
	})
})

var _ = Describe("ModuleNotImportedError", func() {
	It("Should format Error using the alias", func() {
		err := &symbol.ModuleNotImportedError{Alias: "time", Name: "time.now"}
		Expect(err.Error()).To(Equal(`module "time" is not imported`))
	})

	It("Should suggest the import statement in GetHint", func() {
		err := &symbol.ModuleNotImportedError{Alias: "time", Name: "time.now"}
		Expect(err.GetHint()).To(Equal("add `import ( time )` at the top of the file"))
	})
})

var _ = Describe("ListModules", func() {
	It("Should return module names from a ModuleResolver", func() {
		r := &symbol.ModuleResolver{Name: "time"}
		Expect(symbol.ListModules(r)).To(ConsistOf("time"))
	})

	It("Should walk a CompoundResolver and aggregate module names", func() {
		r := symbol.CompoundResolver{
			&symbol.ModuleResolver{Name: "time"},
			&symbol.ModuleResolver{Name: "math"},
		}
		Expect(symbol.ListModules(r)).To(ConsistOf("time", "math"))
	})

	It("Should return an empty slice when no modules are present", func() {
		Expect(symbol.ListModules(symbol.MapResolver{})).To(BeEmpty())
	})
})
