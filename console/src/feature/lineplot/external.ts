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
import { LAYOUT_TYPE } from "@/platform/lineplot/layout";
import { type Selector } from "@/platform/selector";

export * from "@/feature/lineplot/link";
export * from "@/feature/lineplot/ontology";
export * from "@/feature/lineplot/palette";
export * from "@/feature/lineplot/toolbar";
export * from "@/feature/lineplot/useTriggerHold";
export * from "@/platform/lineplot/addChannelsToActivePlot";
export * from "@/platform/lineplot/layout";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = { [LAYOUT_TYPE]: ingest };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: LinePlot };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
export * from "@/platform/lineplot/useCreate";
