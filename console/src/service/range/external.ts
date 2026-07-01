// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Range as ComponentRange } from "@/component/range";
import { type Layout } from "@/component/layout";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/service/range/Explorer";
import { Overview } from "@/service/range/overview/Overview";

export * from "@/service/range/ContextMenu";
export * from "@/service/range/Explorer";
export * from "@/service/range/link";
export * from "@/service/range/list";
export * from "@/service/range/ontology";
export * from "@/service/range/overview/Overview";
export * from "@/service/range/palette";
export * from "@/service/range/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [ComponentRange.OVERVIEW_LAYOUT_TYPE]: Overview,
  [EXPLORER_LAYOUT_TYPE]: Explorer,
};
