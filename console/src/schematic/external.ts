// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { ContextMenu, LAYOUT_TYPE, Schematic, SELECTABLE } from "@/schematic/Schematic";

export * from "@/schematic/export";
export * from "@/schematic/middleware";
export * from "@/schematic/NavControls";
export * from "@/schematic/Schematic";
export * from "@/schematic/selectors";
export * from "@/schematic/slice";
export * from "@/schematic/toolbar";
export * from "@/schematic/useRangeSnapshot";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
};

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const SELECTABLES: Layout.Selectable[] = [SELECTABLE];
