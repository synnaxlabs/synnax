// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { type Layout } from "@/layout";
import { extract } from "@/schematic/export";
import {
  ContextMenu,
  LAYOUT_TYPE,
  Schematic,
  SchematicSelectable,
} from "@/schematic/Schematic";
import { Edit, EDIT_LAYOUT_TYPE } from "@/schematic/symbols/edit/Edit";
import { type Selector } from "@/selector";

export * from "@/schematic/export";
export * from "@/schematic/middleware";
export * from "@/schematic/Schematic";
export * from "@/schematic/selectors";
export * from "@/schematic/slice";
export * from "@/schematic/toolbar";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
  [EDIT_LAYOUT_TYPE]: Edit,
};

export const SELECTABLES: Selector.Selectable[] = [SchematicSelectable];
