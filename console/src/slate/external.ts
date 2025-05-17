// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { type Selector } from "@/selector";
import { ContextMenu, LAYOUT_TYPE, SELECTABLE, Slate } from "@/slate/Slate";

export * from "@/slate/export";
export * from "@/slate/middleware";
export * from "@/slate/NavControls";
export * from "@/slate/selectors";
export * from "@/slate/Slate";
export * from "@/slate/slice";
export * from "@/slate/toolbar";
export * from "@/slate/useRangeSnapshot";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Slate,
};

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const SELECTABLES: Selector.Selectable[] = [SELECTABLE];
