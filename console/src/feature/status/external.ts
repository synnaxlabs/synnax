// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Explorer } from "@/feature/status/explorer";
import { type Panel } from "@/platform/panel";

export * from "@/feature/status/explorer";
export * from "@/feature/status/palette";
export * from "@/feature/status/Toolbar";
export * from "@/platform/status/external";

export const TABS: Panel.Tabs = Explorer.TABS;
