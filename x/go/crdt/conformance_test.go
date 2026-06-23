// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crdt_test

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/crdt"
	. "github.com/synnaxlabs/x/testutil"
)

// The conformance vectors are authored once and shared with the TypeScript
// implementation so that both runtimes materialize identically for the same edit
// schedule. The wire encoding of the operations is owned and tested by oracle codegen.
const vectorsPath = "../../testdata/crdt/vectors.json"

type vectorFile struct {
	Behavior []behaviorVector `json:"behavior"`
}

type behaviorVector struct {
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
	Steps    []step `json:"steps"`
	Expected string `json:"expected"`
}

type step struct {
	Type    string `json:"type"`
	Replica int    `json:"replica"`
	Index   int    `json:"index"`
	Text    string `json:"text"`
	Length  int    `json:"length"`
}

// genOp is a generated operation paired with the replica that produced it, so a sync can
// deliver it to every other replica.
type genOp struct {
	owner int
	apply func(*crdt.Text)
}

// runBehavior executes a behavior vector, returning the converged document of every
// replica.
func runBehavior(v behaviorVector) []*crdt.Text {
	docs := make([]*crdt.Text, v.Replicas)
	for i := range docs {
		docs[i] = crdt.New(uint32(i + 1))
	}
	var generated []genOp
	record := func(replica int, op func(*crdt.Text)) {
		generated = append(generated, genOp{owner: replica - 1, apply: op})
	}
	sync := func() {
		for di, d := range docs {
			for _, g := range generated {
				if g.owner != di {
					g.apply(d)
				}
			}
		}
	}
	for _, s := range v.Steps {
		switch s.Type {
		case "insert":
			ops := docs[s.Replica-1].Insert(s.Index, s.Text)
			record(s.Replica, func(t *crdt.Text) { t.ApplyInsert(ops...) })
		case "delete":
			ops := docs[s.Replica-1].Delete(s.Index, s.Length)
			record(s.Replica, func(t *crdt.Text) { t.ApplyDelete(ops...) })
		case "sync":
			sync()
		}
	}
	sync()
	return docs
}

var _ = Describe("Conformance vectors", func() {
	var file vectorFile
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &file)).To(Succeed())
	for _, v := range file.Behavior {
		It("converges: "+v.Name, func() {
			for _, d := range runBehavior(v) {
				Expect(d.String()).To(Equal(v.Expected))
			}
		})
	}
})
