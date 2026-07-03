// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ContextMenu } from "@/feature/schematic/ContextMenu";
import { extract } from "@/feature/schematic/export";
import { ingest } from "@/feature/schematic/import";
import { Schematic } from "@/feature/schematic/Schematic";
import { Selectable } from "@/feature/schematic/Selectable";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { Schematic as CommonSchematic } from "@/platform/schematic";
import { type Selector } from "@/platform/selector";

export * from "@/feature/schematic/import";
export * from "@/feature/schematic/link";
export * from "@/feature/schematic/ontology";
export * from "@/feature/schematic/palette";
export * from "@/feature/schematic/symbol";
export * from "@/feature/schematic/toolbar/Toolbar";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [CommonSchematic.LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [CommonSchematic.LAYOUT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = {
  [CommonSchematic.LAYOUT_TYPE]: ingest,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CommonSchematic.LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
export * from "@/platform/schematic/external";
