// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/feature/range/Explorer";
import { Overview } from "@/feature/range/overview/Overview";
import { type Layout } from "@/platform/layout";
import { Range } from "@/platform/range";

export * from "@/feature/range/ContextMenu";
export * from "@/feature/range/Explorer";
export * from "@/feature/range/link";
export * from "@/feature/range/list";
export * from "@/feature/range/ontology";
export * from "@/feature/range/overview/Overview";
export * from "@/feature/range/commands";
export * from "@/feature/range/Toolbar";
export * from "@/platform/range/external";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [Range.OVERVIEW_LAYOUT_TYPE]: Overview,
  [EXPLORER_LAYOUT_TYPE]: Explorer,
};
