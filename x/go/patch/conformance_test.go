// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch_test

import (
	"encoding/json/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// The vectors are shared with the TypeScript implementation so that both runtimes apply
// an action to a document alike.
const vectorsPath = "../../testdata/patch/vectors.json"

type vectorFile struct {
	Cases []vector `json:"cases"`
}

type vector struct {
	Name     string         `json:"name"`
	Doc      map[string]any `json:"doc"`
	Actions  []action       `json:"actions"`
	Expected map[string]any `json:"expected"`
	Error    string         `json:"error"`
}

type action struct {
	Type   string        `json:"type"`
	Path   []string      `json:"path"`
	Value  any           `json:"value"`
	Key    string        `json:"key"`
	After  *string       `json:"after"`
	Guards []patch.Guard `json:"guards"`
}

func (a action) apply(doc map[string]any) error {
	GinkgoHelper()
	switch a.Type {
	case "set":
		return patch.Set{Path: a.Path, Value: a.Value, Guards: a.Guards}.Apply(doc)
	case "remove":
		return patch.Remove{Path: a.Path, Key: a.Key, Guards: a.Guards}.Apply(doc)
	case "insert":
		return patch.Insert{
			Path: a.Path, Value: a.Value, After: a.After, Guards: a.Guards,
		}.Apply(doc)
	case "move":
		return patch.Move{
			Path: a.Path, Key: a.Key, After: a.After, Guards: a.Guards,
		}.Apply(doc)
	}
	Fail("unknown action type " + a.Type)
	return nil
}

var _ = Describe("Conformance vectors", func() {
	var file vectorFile
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &file)).To(Succeed())
	for _, v := range file.Cases {
		It(v.Name, func() {
			var err error
			for _, a := range v.Actions {
				if err = a.apply(v.Doc); err != nil {
					break
				}
			}
			if v.Error != "" {
				Expect(err).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(v.Error)),
				))
				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(v.Doc).To(Equal(v.Expected))
		})
	}
})
