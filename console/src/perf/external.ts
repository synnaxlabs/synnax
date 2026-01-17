// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Dashboard } from "@/perf/Dashboard";
import { create, LAYOUT_TYPE } from "@/perf/layout";

export { create, LAYOUT_TYPE } from "@/perf/layout";
export { COMMANDS } from "@/perf/palette";
export * from "@/perf/selectors";
export * from "@/perf/slice";

/** Layout renderers for the performance module. */
export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Dashboard,
};

/** Create a performance dashboard layout. */
export const createDashboard = create;
