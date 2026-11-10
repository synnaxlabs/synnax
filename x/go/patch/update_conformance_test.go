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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// The vectors are shared with the TypeScript implementation so that both runtimes apply
// an update alike.
const updateVectorsPath = "../../testdata/update/vectors.json"

// testDoc and its update stand for a generated record and its generated Apply. A
// channel is a union on type: "ai" has a range and units with defaults, and "di" has a
// line without one. Limits have a low with a default, and a high without one. Cells are
// a map of structs.
type testDoc struct {
	Name   string              `json:"name"`
	Group  *string             `json:"group,omitzero"`
	Config testConfig          `json:"config"`
	Limits *testLimits         `json:"limits,omitzero"`
	Tags   []string            `json:"tags"`
	Cells  map[string]testCell `json:"cells,omitzero"`
}

type testCell struct {
	Value string `json:"value"`
	Width int    `json:"width"`
}

type testConfig struct {
	Rate     int           `json:"rate"`
	Channels []testChannel `json:"channels"`
}

type testChannel struct {
	Key   string  `json:"key"`
	Type  string  `json:"type"`
	Port  int     `json:"port"`
	Range *int    `json:"range,omitzero"`
	Units *string `json:"units,omitzero"`
	Line  *int    `json:"line,omitzero"`
}

type testLimits struct {
	Low  int `json:"low"`
	High int `json:"high"`
}

type testLimitsField = patch.Struct[testLimitsUpdate, testLimits]

type testDocUpdate struct {
	Name   patch.Change[string]                             `json:"name,omitzero"`
	Group  patch.Change[string]                             `json:"group,omitzero"`
	Config patch.Struct[testConfigUpdate, testConfig]       `json:"config,omitzero"`
	Limits patch.Change[testLimitsField]                    `json:"limits,omitzero"`
	Tags   patch.ValueList[string]                          `json:"tags,omitzero"`
	Cells  patch.TypedMap[string, testCell, testCellUpdate] `json:"cells,omitzero"`
}

type testCellUpdate struct {
	Value patch.Change[string] `json:"value,omitzero"`
	Width patch.Change[int]    `json:"width,omitzero"`
}

type testConfigUpdate struct {
	Rate     patch.Change[int]                                  `json:"rate,omitzero"`
	Channels patch.List[testChannel, string, testChannelUpdate] `json:"channels,omitzero"`
}

type testChannelUpdate struct {
	Key   string               `json:"key"`
	Type  string               `json:"type"`
	Port  patch.Change[int]    `json:"port,omitzero"`
	Range patch.Change[int]    `json:"range,omitzero"`
	Units patch.Change[string] `json:"units,omitzero"`
	Line  patch.Change[int]    `json:"line,omitzero"`
}

type testLimitsUpdate struct {
	Low  patch.Change[int] `json:"low,omitzero"`
	High patch.Change[int] `json:"high,omitzero"`
}

func (u testDocUpdate) apply(d *testDoc) error {
	if err := u.Name.ApplyTo(&d.Name); err != nil {
		return err
	}
	u.Group.ApplyOptional(&d.Group)
	if err := u.Config.ApplyTo(&d.Config, mergeTestConfig); err != nil {
		return err
	}
	if err := patch.ApplyOptionalStruct(
		&d.Limits,
		u.Limits,
		mergeTestLimits,
		testLimitsDefaults,
	); err != nil {
		return err
	}
	if err := patch.ApplyValueList(&d.Tags, u.Tags); err != nil {
		return err
	}
	return patch.ApplyMap(&d.Cells, u.Cells, mergeTestCell)
}

func mergeTestCell(c *testCell, u testCellUpdate) error {
	if err := u.Value.ApplyTo(&c.Value); err != nil {
		return err
	}
	return u.Width.ApplyTo(&c.Width)
}

func mergeTestConfig(c *testConfig, u testConfigUpdate) error {
	if err := u.Rate.ApplyTo(&c.Rate); err != nil {
		return err
	}
	return patch.ApplyList(
		&c.Channels,
		u.Channels,
		func(c testChannel) string { return c.Key },
		func(u testChannelUpdate) string { return u.Key },
		mergeTestChannel,
	)
}

func testLimitsDefaults(u testLimitsUpdate) (testLimits, error) {
	if u.High.IsZero() {
		return testLimits{}, errors.Wrap(validate.ErrValidation, "high is required")
	}
	return testLimits{Low: 0}, nil
}

func mergeTestLimits(l *testLimits, u testLimitsUpdate) error {
	if err := u.Low.ApplyTo(&l.Low); err != nil {
		return err
	}
	return u.High.ApplyTo(&l.High)
}

// applyRequired applies c to a required variant field that dst points to, filling an
// absent field with fallback, or failing when there is no fallback.
func applyRequired[T any](dst **T, c patch.Change[T], name string, fallback *T) error {
	var v T
	switch {
	case *dst != nil:
		v = **dst
	case !c.IsZero():
	case fallback != nil:
		v = *fallback
	default:
		return errors.Wrapf(validate.ErrValidation, "%s is required", name)
	}
	if err := c.ApplyTo(&v); err != nil {
		return err
	}
	*dst = &v
	return nil
}

func mergeTestChannel(c *testChannel, u testChannelUpdate) error {
	next := testChannel{Key: c.Key, Type: u.Type, Port: c.Port}
	if u.Type == c.Type {
		next = *c
	}
	if err := u.Port.ApplyTo(&next.Port); err != nil {
		return err
	}
	switch u.Type {
	case "ai":
		defaultRange, defaultUnits := 10, "V"
		if err := applyRequired(
			&next.Range,
			u.Range,
			"range",
			&defaultRange,
		); err != nil {
			return err
		}
		if err := applyRequired(
			&next.Units,
			u.Units,
			"units",
			&defaultUnits,
		); err != nil {
			return err
		}
	case "di":
		if err := applyRequired(&next.Line, u.Line, "line", nil); err != nil {
			return err
		}
	default:
		return errors.Wrapf(validate.ErrValidation, "type %s names no variant", u.Type)
	}
	*c = next
	return nil
}

type updateVector struct {
	Name     string                      `json:"name"`
	Doc      jsontext.Value              `json:"doc"`
	Update   patch.Update[testDocUpdate] `json:"update"`
	Expected jsontext.Value              `json:"expected"`
	Error    *struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
	} `json:"error"`
}

var updateErrorKinds = map[string]error{
	"stale":      patch.ErrStale,
	"unique":     query.ErrUniqueViolation,
	"validation": validate.ErrValidation,
}

var _ = Describe("Update conformance vectors", func() {
	var file struct {
		Doc   jsontext.Value `json:"doc"`
		Cases []updateVector `json:"cases"`
	}
	Expect(json.Unmarshal(
		MustSucceed(os.ReadFile(updateVectorsPath)),
		&file,
	)).To(Succeed())
	for _, v := range file.Cases {
		It(v.Name, func() {
			data := v.Doc
			if data == nil {
				data = file.Doc
			}
			var doc testDoc
			Expect(json.Unmarshal(data, &doc)).To(Succeed())
			err := v.Update.Check(doc)
			next := doc
			if err == nil {
				err = v.Update.Value.apply(&next)
			}
			if v.Error != nil {
				Expect(err).To(SatisfyAll(
					MatchError(updateErrorKinds[v.Error.Kind]),
					MatchError(ContainSubstring(v.Error.Message)),
				))
				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(json.Marshal(next)).To(MatchJSON(v.Expected))
			Expect(json.Marshal(doc)).To(MatchJSON(data))
		})
	}
})
