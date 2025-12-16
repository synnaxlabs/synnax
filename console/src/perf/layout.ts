// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";

/** Layout type identifier for the performance dashboard. */
export const LAYOUT_TYPE = "perfDashboard";

/** Create a layout state for the performance dashboard. */
export const create = (
  overrides: Partial<Layout.BaseState> = {},
): Layout.BaseState => ({
  key: LAYOUT_TYPE,
  type: LAYOUT_TYPE,
  name: "Performance Dashboard",
  location: "mosaic",
  ...overrides,
});
