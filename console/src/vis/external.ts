// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { LAYOUT_SELECTOR_TYPE, LayoutSelector } from "@/vis/LayoutSelector";

export * from "@/vis/Canvas";
export * from "@/vis/LayoutSelector";
export * from "@/vis/NavControls";
export * from "@/vis/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_SELECTOR_TYPE]: LayoutSelector,
};
