// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot/LinePlot";
import { LAYOUT_TYPE } from "@/lineplot/slice";

export * from "@/lineplot/LinePlot";
export * from "@/lineplot/link";
export * from "@/lineplot/middleware";
export * from "@/lineplot/NavControls";
export * from "@/lineplot/ontology";
export * from "@/lineplot/palette";
export * from "@/lineplot/selectors";
export * from "@/lineplot/slice";
export * from "@/lineplot/toolbar";
export * from "@/lineplot/useTriggerHold";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: LinePlot,
};
