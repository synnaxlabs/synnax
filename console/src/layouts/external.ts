// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Main, MAIN_LAYOUT_TYPE } from "@/layouts/Main";
import { Mosaic, MOSAIC_LAYOUT_TYPE } from "@/layouts/Mosaic";

export * from "@/layouts/nav";
export * from "@/layouts/Notifications";
export * from "@/layouts/useTriggers";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [MAIN_LAYOUT_TYPE]: Main,
  [MOSAIC_LAYOUT_TYPE]: Mosaic,
};
