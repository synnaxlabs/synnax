// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ContextMenu } from "@/feature/lineplot/ContextMenu";
import { extract } from "@/feature/lineplot/export";
import { ingest } from "@/feature/lineplot/import";
import { LinePlot } from "@/feature/lineplot/LinePlot";
import { Selectable } from "@/feature/lineplot/Selectable";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { LinePlot as PlatformLinePlot } from "@/platform/lineplot";
import { type Selector } from "@/platform/selector";

export * from "@/feature/lineplot/import";
export * from "@/feature/lineplot/link";
export * from "@/feature/lineplot/useMosaicDrop";
export * from "@/feature/lineplot/tree";
export * from "@/feature/lineplot/search";
export * from "@/feature/lineplot/commands";
export * from "@/feature/lineplot/toolbar";
export * from "@/feature/lineplot/useTriggerHold";
export * from "@/platform/lineplot/external";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [PlatformLinePlot.LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = {
  [PlatformLinePlot.LAYOUT_TYPE]: extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [PlatformLinePlot.LAYOUT_TYPE]: ingest,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [PlatformLinePlot.LAYOUT_TYPE]: LinePlot,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
