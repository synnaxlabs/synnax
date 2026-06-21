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
import { Schematic } from "@/schematic/body/Schematic";
import { ContextMenu } from "@/schematic/ContextMenu";
import { extract } from "@/schematic/imex/export";
import { LAYOUT_TYPE } from "@/schematic/layout";
import { Selectable } from "@/schematic/Selectable";
import { Edit, EDIT_LAYOUT_TYPE } from "@/schematic/symbols/edit/Edit";
import { type Selector } from "@/selector";

export * from "@/schematic/imex";
export * from "@/schematic/layout";
export * from "@/schematic/link";
export * from "@/schematic/ontology";
export * from "@/schematic/palette";
export * from "@/schematic/session";
export * from "@/schematic/toolbar";
export * from "@/schematic/useCreate";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
  [EDIT_LAYOUT_TYPE]: Edit,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
