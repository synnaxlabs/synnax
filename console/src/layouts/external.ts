// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { GET_STARTED_LAYOUT_TYPE } from "@/layout/slice";
import { GetStarted } from "@/layouts/GetStarted";
import { Main, MAIN_TYPE } from "@/layouts/Main";
import { Mosaic, MOSAIC_TYPE, MosaicWindow } from "@/layouts/Mosaic";
import { Selector, SELECTOR_TYPE } from "@/layouts/Selector";
export { createSelector } from "@/layouts/Selector";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [MAIN_TYPE]: Main,
  [SELECTOR_TYPE]: Selector,
  [MOSAIC_TYPE]: Mosaic,
  [GET_STARTED_LAYOUT_TYPE]: GetStarted,
  [Layout.MOSAIC_WINDOW_TYPE]: MosaicWindow,
};
