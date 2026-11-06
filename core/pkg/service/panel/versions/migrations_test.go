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
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
)

var _ = Describe("Migrations", func() {
	// Applied migrations are tracked by key, so renaming or reordering an entry
	// re-runs or skips it on stores that already migrated. The chain is pinned
	// verbatim; extend it for a new version, never edit shipped entries.
	It("Should pin the migration chain keys in order", func() {
		Expect(lo.Map(
			slices.Concat(versions.Migrations, versions.CompositionMigrations),
			func(m gorp.Migration, _ int) string { return m.Key() },
		)).To(Equal([]string{
			"msgpack_to_orc",
			"v56_migrate_project_layouts_to_panels",
			"v56_task_tab_uuid_keys",
			"v56_strip_invalid_resource_tabs",
			"v60_drop_view_args",
			"v60_add_icon_and_color",
			"v61_color_alpha_byte",
		}))
	})
})
