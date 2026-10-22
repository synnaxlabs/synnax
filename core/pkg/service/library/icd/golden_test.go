// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
	. "github.com/synnaxlabs/x/testutil"
)

// goldenDir holds the DBC files whose imported libraries the Driver codec tests decode
// against cantools. SYNNAX_GOLDEN_DIR replaces it, as for a downloaded corpus.
func goldenDir() string {
	if dir := os.Getenv("SYNNAX_GOLDEN_DIR"); dir != "" {
		return dir
	}
	return filepath.FromSlash("../../../../../driver/codec/testdata/dbc")
}

func goldenEntries() []TableEntry {
	paths := MustSucceed(filepath.Glob(filepath.Join(goldenDir(), "*.dbc")))
	entries := make([]TableEntry, 0, len(paths))
	for _, p := range paths {
		entries = append(entries, Entry(filepath.Base(p), p))
	}
	return entries
}

// deterministicKeys replaces every entry and field key with one numbered by its order
// in the library, so that the same DBC always imports to the same JSON.
func deterministicKeys(entries []library.Entry) []library.Entry {
	GinkgoHelper()
	keys := make(map[uuid.UUID]uuid.UUID)
	next := func(old uuid.UUID) uuid.UUID {
		k := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", len(keys)+1))
		keys[old] = k
		return k
	}
	ref := func(old *uuid.UUID) *uuid.UUID {
		if old == nil {
			return nil
		}
		k, ok := keys[*old]
		Expect(ok).To(BeTrue(), "key %s names no entry or field", *old)
		return &k
	}
	out := make([]library.Entry, len(entries))
	for i, e := range entries {
		switch v := e.Variant.(type) {
		case library.EnumEntry:
			v.Key = next(v.Key)
			out[i] = library.Entry{Variant: v}
		case library.MessageEntry:
			v.Key = next(v.Key)
			p := v.Payload.Variant.(library.BinaryPayload)
			p.Fields = append([]library.BinaryField{}, p.Fields...)
			for j := range p.Fields {
				p.Fields[j].Key = next(p.Fields[j].Key)
			}
			v.Payload = library.Payload{Variant: p}
			out[i] = library.Entry{Variant: v}
		}
	}
	for i, e := range out {
		v, ok := e.Variant.(library.MessageEntry)
		if !ok {
			continue
		}
		p := v.Payload.Variant.(library.BinaryPayload)
		for j := range p.Fields {
			p.Fields[j].Multiplexor = ref(p.Fields[j].Multiplexor)
			p.Fields[j].Enumeration = ref(p.Fields[j].Enumeration)
		}
		out[i] = library.Entry{Variant: v}
	}
	return out
}

var _ = Describe("Golden libraries", func() {
	DescribeTable(
		"Should import each DBC to its golden library JSON",
		func(path string) {
			data := MustSucceed(os.ReadFile(path))
			entries := MustSucceed(icd.Parse(icd.FormatDBC, data))
			name := strings.TrimSuffix(filepath.Base(path), ".dbc")
			lib := library.Library{
				Key:     uuid.MustParse("00000000-0000-4000-8000-000000000000"),
				Name:    name,
				Entries: deterministicKeys(entries),
			}
			got := MustSucceed(json.Marshal(lib, jsontext.WithIndent("  ")))
			got = append(got, '\n')
			golden := strings.TrimSuffix(path, ".dbc") + ".library.json"
			if os.Getenv("SYNNAX_UPDATE_GOLDEN") == "1" {
				Expect(os.WriteFile(golden, got, 0o644)).To(Succeed())
				return
			}
			Expect(string(got)).To(Equal(string(MustSucceed(os.ReadFile(golden)))))
		},
		goldenEntries(),
	)
})
