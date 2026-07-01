// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/component/layout";
import { ContextMenu } from "@/component/lineplot/ContextMenu";
import { LAYOUT_TYPE } from "@/component/lineplot/layout";
import { LinePlot } from "@/component/lineplot/LinePlot";
import { type Selector } from "@/component/selector";
import { type Export } from "@/service/export";
import { extract } from "@/service/lineplot/export";
import { Selectable } from "@/service/lineplot/Selectable";

export * from "@/component/lineplot/layout";
export * from "@/component/lineplot/toolbar";
export * from "@/service/lineplot/addChannelsToActivePlot";
export * as ImEx from "@/service/lineplot/imex";
export * from "@/service/lineplot/link";
export * from "@/service/lineplot/ontology";
export * from "@/service/lineplot/palette";
export * from "@/service/lineplot/useCreate";
export * from "@/service/lineplot/useTriggerHold";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: LinePlot };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
