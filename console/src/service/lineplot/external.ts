// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { LinePlot } from "@/service/lineplot/body/LinePlot";
import { ContextMenu } from "@/service/lineplot/ContextMenu";
import { extract } from "@/service/lineplot/imex/export";
import { LAYOUT_TYPE } from "@/service/lineplot/layout";
import { Selectable } from "@/service/lineplot/Selectable";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/service/lineplot/addChannelsToActivePlot";
export * from "@/service/lineplot/imex";
export * from "@/service/lineplot/layout";
export * from "@/service/lineplot/link";
export * from "@/service/lineplot/ontology";
export * from "@/service/lineplot/palette";
export * from "@/service/lineplot/toolbar";
export * from "@/service/lineplot/useCreate";
export * from "@/service/lineplot/useTriggerHold";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: LinePlot };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
