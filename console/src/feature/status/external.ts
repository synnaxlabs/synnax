// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/feature/status/Explorer";
import { type Layout } from "@/platform/layout";

export * from "@/feature/status/Explorer";
export * from "@/feature/status/commands";
export * from "@/feature/status/Toolbar";
export * from "@/platform/status/external";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EXPLORER_LAYOUT_TYPE]: Explorer,
};
