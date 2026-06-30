// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Schematic } from "@/service/schematic/body/Schematic";
import { ContextMenu } from "@/service/schematic/ContextMenu";
import { extract } from "@/service/schematic/imex/export";
import { LAYOUT_TYPE } from "@/service/schematic/layout";
import { Selectable } from "@/service/schematic/Selectable";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/service/schematic/imex";
export * from "@/service/schematic/layout";
export * from "@/service/schematic/link";
export * from "@/service/schematic/ontology";
export * from "@/service/schematic/palette";
export * from "@/service/schematic/toolbar";
export * from "@/service/schematic/useCreate";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
