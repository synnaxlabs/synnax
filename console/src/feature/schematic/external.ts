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
import { Schematic as PlatformSchematic } from "@/platform/schematic";
import { type Selector } from "@/platform/selector";

export * from "@/feature/schematic/import";
export * from "@/feature/schematic/link";
export * from "@/feature/schematic/useMosaicDrop";
export * from "@/feature/schematic/tree";
export * from "@/feature/schematic/search";
export * from "@/feature/schematic/commands";
export * from "@/feature/schematic/symbol";
export * from "@/feature/schematic/toolbar/Toolbar";
export * from "@/platform/schematic/external";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [PlatformSchematic.LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = {
  [PlatformSchematic.LAYOUT_TYPE]: extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [PlatformSchematic.LAYOUT_TYPE]: ingest,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [PlatformSchematic.LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
