// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Modals } from "@/modals";
import { Create, CREATE_LAYOUT_TYPE } from "@/status/Create";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/status/Explorer";
import { type Tabs } from "@/panel/tabs/index";

export * from "@/status/Create";
export * from "@/status/Explorer";
export * from "@/status/palette";
export * from "@/status/slice";
export * from "@/status/Toolbar";
export * from "@/status/useListenForChanges";

export const MODALS: Record<string, Modals.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
};

export const TABS: Record<string, Tabs.Renderer> = {
  [EXPLORER_LAYOUT_TYPE]: Explorer,
};
