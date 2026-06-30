// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Schematic } from "@/component/schematic/body/Schematicchematic";
import { LAYOUT_TYPE } from "@/component/schematic/layouts/layout";
import { type Export } from "@/export";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";
import { extract } from "@/service/schematic/export";
import { Selectable } from "@/service/schematic/Selectable";
import { ContextMenu } from "@/service/schematic/symbols/ContextMenu";

export * from "@/component/schematic/layouts/layout";
export * from "@/component/schematic/useCreateseCreate";
export * from "@/service/schematic/imex";
export * from "@/service/schematic/link";
export * from "@/service/schematic/ontology";
export * from "@/service/schematic/palette";
export * from "@/service/schematic/symbols/toolbar";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
