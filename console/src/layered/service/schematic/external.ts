// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Schematic } from "@/layered/service/schematic/body/Schematic";
import { ContextMenu } from "@/layered/service/schematic/ContextMenu";
import { extract } from "@/layered/service/schematic/imex/export";
import { LAYOUT_TYPE } from "@/layered/service/schematic/layout";
import { Selectable } from "@/layered/service/schematic/Selectable";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/layered/service/schematic/imex";
export * from "@/layered/service/schematic/layout";
export * from "@/layered/service/schematic/link";
export * from "@/layered/service/schematic/ontology";
export * from "@/layered/service/schematic/palette";
export * from "@/layered/service/schematic/toolbar";
export * from "@/layered/service/schematic/useCreate";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
