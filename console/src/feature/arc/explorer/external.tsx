// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Explorer } from "@/feature/arc/explorer/Explorer";
import { Panel } from "@/platform/panel";

export { Explorer };

export const TAB_TYPE = "arc_explorer";

const TAB: Panel.Tab = {
  Content: Explorer,
  Name: Panel.createStaticTabName({ name: "Arc Explorer", icon: <Icon.Arc /> }),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};

export const useOpenTab = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return () => openTab({ variant: "view", type: TAB_TYPE, args: {} });
};
