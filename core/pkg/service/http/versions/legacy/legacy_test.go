// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/http/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Read", func() {
	It("Should convert v0 records to lists and flip fields", func() {
		in := msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{
				"headers":     map[string]any{"Accept": "application/json"},
				"queryParams": map[string]any{"limit": "10"},
				"fields": []any{map[string]any{
					"enabled":    false,
					"enumValues": map[string]any{"ON": float64(1)},
				}},
			}},
		}
		Expect(legacy.Read.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{
				"headers": []any{map[string]any{
					"name": "Accept", "value": "application/json",
				}},
				"query_params": []any{map[string]any{
					"parameter": "limit", "value": "10",
				}},
				"fields": []any{map[string]any{
					"disabled": true,
					"enum_values": []any{map[string]any{
						"label": "ON", "value": float64(1),
					}},
				}},
			}},
		}))
	})

	It("Should keep record keys that look like camelCase as data", func() {
		in := msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{
				"headers": map[string]any{"xApiKey": "secret"},
			}},
		}
		Expect(legacy.Read.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{
				"headers": []any{map[string]any{
					"name": "xApiKey", "value": "secret",
				}},
			}},
		}))
	})
})

var _ = Describe("Write", func() {
	It("Should flip v0 endpoints and convert records", func() {
		in := msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{
				"enabled":     true,
				"queryParams": map[string]any{"limit": "10"},
			}},
		}
		Expect(legacy.Write.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{
				"disabled": false,
				"query_params": []any{map[string]any{
					"parameter": "limit", "value": "10",
				}},
			}},
		}))
	})
})
