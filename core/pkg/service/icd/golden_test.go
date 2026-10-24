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
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	. "github.com/synnaxlabs/x/testutil"
)

// goldenDir holds the DBC files whose imported ICDs the Driver codec tests decode
// against cantools. SYNNAX_GOLDEN_DIR replaces it, as for a downloaded corpus.
func goldenDir() string {
	if dir := os.Getenv("SYNNAX_GOLDEN_DIR"); dir != "" {
		return dir
	}
	return filepath.FromSlash("../../../../driver/codec/testdata/dbc")
}

func goldenEntries() []TableEntry {
	paths := MustSucceed(filepath.Glob(filepath.Join(goldenDir(), "*.dbc")))
	Expect(paths).ToNot(BeEmpty(), "no DBC files in %s", goldenDir())
	entries := make([]TableEntry, 0, len(paths))
	for _, p := range paths {
		entries = append(entries, Entry(filepath.Base(p), p))
	}
	return entries
}

// deterministicKeys replaces every message and field key with one numbered by its order
// in the ICD, so that the same DBC always imports to the same JSON.
func deterministicKeys(messages []icd.Message) []icd.Message {
	GinkgoHelper()
	keys := make(map[uuid.UUID]uuid.UUID)
	next := func(old uuid.UUID) uuid.UUID {
		k := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", len(keys)+1))
		keys[old] = k
		return k
	}
	out := make([]icd.Message, len(messages))
	for i, m := range messages {
		m.Key = next(m.Key)
		p := m.Payload.Variant.(icd.BinaryPayload)
		p.Fields = append([]icd.BinaryField{}, p.Fields...)
		for j := range p.Fields {
			p.Fields[j].Key = next(p.Fields[j].Key)
		}
		for j := range p.Fields {
			if mux := p.Fields[j].Multiplexor; mux != nil {
				k, ok := keys[*mux]
				Expect(ok).To(BeTrue(), "multiplexor %s names no field", *mux)
				p.Fields[j].Multiplexor = &k
			}
		}
		m.Payload = icd.Payload{Variant: p}
		out[i] = m
	}
	return out
}

// SYNNAX_UPDATE_GOLDEN=1 rewrites the golden files. Format them with Prettier after.
var _ = Describe("Golden ICDs", func() {
	DescribeTable(
		"Should import each DBC to its golden ICD JSON",
		func(path string) {
			data := MustSucceed(os.ReadFile(path))
			messages := MustSucceed(icd.Parse(icd.FormatDBC, data))
			name := strings.TrimSuffix(filepath.Base(path), ".dbc")
			ic := icd.ICD{
				Key:      uuid.MustParse("00000000-0000-4000-8000-000000000000"),
				Name:     name,
				Messages: deterministicKeys(messages),
			}
			got := MustSucceed(json.Marshal(ic, jsontext.WithIndent("  ")))
			golden := strings.TrimSuffix(path, ".dbc") + ".icd.json"
			if os.Getenv("SYNNAX_UPDATE_GOLDEN") == "1" {
				Expect(os.WriteFile(golden, got, 0o644)).To(Succeed())
				return
			}
			Expect(got).To(MatchJSON(MustSucceed(os.ReadFile(golden))))
		},
		goldenEntries(),
	)
})
