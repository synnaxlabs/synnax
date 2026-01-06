// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { GetStarted } from "@/layouts/GetStarted";
import { Main, MAIN_LAYOUT_TYPE } from "@/layouts/Main";
import { Mosaic, MOSAIC_LAYOUT_TYPE, MosaicWindow } from "@/layouts/Mosaic";
import { Selector, SELECTOR_LAYOUT_TYPE } from "@/layouts/Selector";

export * from "@/layouts/nav";
export * from "@/layouts/Notifications";
export * from "@/layouts/useTriggers";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [MAIN_LAYOUT_TYPE]: Main,
  [SELECTOR_LAYOUT_TYPE]: Selector,
  [MOSAIC_LAYOUT_TYPE]: Mosaic,
  [Layout.GET_STARTED_TYPE]: GetStarted,
  [Layout.MOSAIC_WINDOW_TYPE]: MosaicWindow,
};
