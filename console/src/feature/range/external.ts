// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Explorer } from "@/feature/range/explorer";
import { Overview } from "@/feature/range/overview";
import { type Panel } from "@/platform/panel";

export * from "@/feature/range/ContextMenu";
export * from "@/feature/range/explorer";
export * from "@/feature/range/link";
export * from "@/feature/range/list";
export * from "@/feature/range/ontology";
export * from "@/feature/range/overview";
export * from "@/feature/range/palette";
export * from "@/feature/range/Toolbar";
export * from "@/platform/range/external";

export const TABS: Panel.Tabs = {
  ...Overview.TABS,
  ...Explorer.TABS,
};
