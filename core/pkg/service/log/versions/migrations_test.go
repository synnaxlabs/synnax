// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
)

var _ = Describe("Migrations", func() {
	// Applied migrations are tracked by key, so renaming or reordering an entry
	// re-runs or skips it on stores that already migrated. The chain is pinned
	// verbatim; extend it for a new version, never edit shipped entries.
	It("Should pin the migration chain keys in order", func() {
		Expect(lo.Map(
			versions.Migrations,
			func(m gorp.Migration, _ int) string { return m.Key() },
		)).To(Equal([]string{
			"normalize_keys",
			"msgpack_to_orc",
			"v55_lift_typed_log",
			"v61_optional_channel_precision",
			"v61_log_group_membership",
			"v61_log_owner",
			"v62_log_delete_range_owned",
			"v62_log_drop_range",
			"v61_color_alpha_byte",
		}))
	})
})
