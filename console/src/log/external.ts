// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { LAYOUT_TYPE, Log, SELECTABLE } from "@/log/Log";

export * from "@/log/export";
export * from "@/log/Log";
export * from "@/log/selectors";
export * from "@/log/slice";
export * from "@/log/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Log,
};

export const SELECTABLES: Layout.Selectable[] = [SELECTABLE];
