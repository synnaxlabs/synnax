// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { Panel } from "@/platform/panel";

export const TAB_TYPE = "settings";

const TAB: Panel.Tab = {
  Icon: Icon.Settings,
  Name: Panel.createStaticTabName({ name: "Settings", icon: <Icon.Settings /> }),
  Content: Panel.lazyComponent(() => import("@/app/settings/Settings"), "Settings"),
};

export const TABS: Panel.Tabs = { [TAB_TYPE]: TAB };

/** Returns a function that opens the Settings tab, or selects it when it is open. */
export const useOpen = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return () => openTab({ variant: "view", type: TAB_TYPE }, { singleton: true });
};
