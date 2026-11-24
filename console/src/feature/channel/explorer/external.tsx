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
import { View } from "@/platform/view";

export const TAB_TYPE = "channel_explorer";

export const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(
    () => import("@/feature/channel/explorer/Explorer"),
    "Explorer",
  ),
  Icon: Icon.Channel,
  Name: Panel.createStaticTabName({
    name: "Channel explorer",
    icon: <Icon.Channel />,
  }),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};

export const useOpenTab = (): (() => void) =>
  View.useOpenExplorer({ resourceType: "channel", tabType: TAB_TYPE });
