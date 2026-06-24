// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { LinePlot } from "@/layered/service/lineplot/body/LinePlot";
import { ContextMenu } from "@/layered/service/lineplot/ContextMenu";
import { extract } from "@/layered/service/lineplot/imex/export";
import { LAYOUT_TYPE } from "@/layered/service/lineplot/layout";
import { Selectable } from "@/layered/service/lineplot/Selectable";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/layered/service/lineplot/addChannelsToActivePlot";
export * from "@/layered/service/lineplot/imex";
export * from "@/layered/service/lineplot/layout";
export * from "@/layered/service/lineplot/link";
export * from "@/layered/service/lineplot/ontology";
export * from "@/layered/service/lineplot/palette";
export * from "@/layered/service/lineplot/toolbar";
export * from "@/layered/service/lineplot/useCreate";
export * from "@/layered/service/lineplot/useTriggerHold";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: LinePlot };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
