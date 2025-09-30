// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EDIT_LAYOUT_TYPE, Editor, SELECTABLE } from "@/arc/Editor";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/arc/Explorer";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/arc/Editor";
export * from "@/arc/Explorer";
export * from "@/arc/export";
export * from "@/arc/middleware";
export * from "@/arc/NavControls";
export * from "@/arc/selectors";
export * from "@/arc/slice";
export * from "@/arc/toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EXPLORER_LAYOUT_TYPE]: Explorer,
  [EDIT_LAYOUT_TYPE]: Editor,
};

export const SELECTABLES: Selector.Selectable[] = [SELECTABLE];
