// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Create, CREATE_LAYOUT_TYPE } from "@/range/Create";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/range/Explorer";
import { OVERVIEW_LAYOUT_TYPE } from "@/range/overview/layout";
import { Overview } from "@/range/overview/Overview";

export * from "@/range/ContextMenu";
export * from "@/range/Create";
export * from "@/range/Explorer";
export * from "@/range/overview/layout";
export * from "@/range/overview/Overview";
export * from "@/range/Select";
export * from "@/range/selectors";
export * from "@/range/slice";
export * from "@/range/slice";
export * from "@/range/Toolbar";
export * from "@/range/translate";
export * from "@/range/useAddToActivePlot";
export * from "@/range/useAddToNewPlot";
export * from "@/range/useListenForChanges";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
  [OVERVIEW_LAYOUT_TYPE]: Overview,
  [EXPLORER_LAYOUT_TYPE]: Explorer,
};
