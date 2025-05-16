// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { ContextMenu, LAYOUT_TYPE, Stage, SELECTABLE } from "@/stage/Stage";
import { type Selector } from "@/selector";

export * from "@/stage/export";
export * from "@/stage/middleware";
export * from "@/stage/NavControls";
export * from "@/stage/Stage";
export * from "@/stage/selectors";
export * from "@/stage/slice";
export * from "@/stage/toolbar";
export * from "@/stage/useRangeSnapshot";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Stage,
};

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const SELECTABLES: Selector.Selectable[] = [SELECTABLE];
