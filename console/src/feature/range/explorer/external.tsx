// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Explorer } from "@/feature/range/explorer/Explorer";
import { Panel } from "@/platform/panel";

export { Explorer };

export const TAB: Panel.Tab = {
  Name: Panel.createStaticTabName({ name: "Range Explorer", icon: <Icon.Explore /> }),
  Content: Explorer,
};

export const TAB_TYPE = "range_explorer";

export const TABS: Panel.Tabs = { [TAB_TYPE]: TAB };

export const useOpenTab = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return () => openTab({ variant: "view", type: TAB_TYPE, args: {} });
};
