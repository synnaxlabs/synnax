// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { type Layout } from "@/layout";
import { extract } from "@/lineplot/export";
import { LAYOUT_TYPE } from "@/lineplot/layout";
import { LinePlot } from "@/lineplot/LinePlot";
import { SELECTABLE } from "@/lineplot/Selectable";
import { type Selector } from "@/selector";

export * from "@/lineplot/export";
export * from "@/lineplot/layout";
export * from "@/lineplot/LinePlot";
export * from "@/lineplot/middleware";
export * from "@/lineplot/NavControls";
export * from "@/lineplot/selectors";
export * from "@/lineplot/slice";
export * from "@/lineplot/toolbar";
export * from "@/lineplot/useTriggerHold";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: LinePlot };

export const SELECTABLES: Selector.Selectable[] = [SELECTABLE];
