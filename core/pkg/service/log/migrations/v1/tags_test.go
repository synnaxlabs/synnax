// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
)

// A log envelope is exported as JSON, YAML, or TOML, and each codec reads its own struct
// tag (json.v3 ignores json tags; go-toml emits the verbatim field name). These specs
// guard against the three drifting: every frozen struct that can be serialized must carry
// identical json, yaml, and toml tags on every field.
var _ = Describe("Encoding tag drift", func() {
	DescribeTable("json, yaml, and toml tags agree on every field",
		func(typ reflect.Type) {
			for i := range typ.NumField() {
				f := typ.Field(i)
				j := f.Tag.Get("json")
				Expect(j).ToNot(BeEmpty(), "field %s has no json tag", f.Name)
				Expect(f.Tag.Get("yaml")).To(Equal(j), "field %s: yaml tag != json tag", f.Name)
				Expect(f.Tag.Get("toml")).To(Equal(j), "field %s: toml tag != json tag", f.Name)
			}
		},
		Entry("Data", reflect.TypeOf(v1.Data{})),
		Entry("ChannelEntry", reflect.TypeOf(v1.ChannelEntry{})),
	)
})
