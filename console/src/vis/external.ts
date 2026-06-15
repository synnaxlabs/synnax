// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Selector, SELECTOR_LAYOUT_TYPE } from "@/vis/Selector";

export * from "@/vis/Canvas";
export * from "@/vis/Selector";
export * from "@/vis/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [SELECTOR_LAYOUT_TYPE]: Selector,
};
