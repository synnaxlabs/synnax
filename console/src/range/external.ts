// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/range/Explorer";
import { OVERVIEW_LAYOUT_TYPE } from "@/range/overview/layout";
import { Overview } from "@/range/overview/Overview";

export * from "@/range/ContextMenu";
export * from "@/range/Create";
export * from "@/range/Explorer";
export * from "@/range/overview/layout";
export * from "@/range/overview/Overview";
export * from "@/range/Select";
export * from "@/layered/session/range/selectors";
export * from "@/layered/session/range/slice";
export * from "@/layered/session/range/slice";
export * from "@/range/Toolbar";
export * from "@/range/translate";
export * from "@/range/useAddToActivePlot";
export * from "@/range/useAddToNewPlot";
export * from "@/range/useListenForChanges";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [OVERVIEW_LAYOUT_TYPE]: Overview,
  [EXPLORER_LAYOUT_TYPE]: Explorer,
};
