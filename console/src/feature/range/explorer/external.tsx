// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Ranger } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";
import { View } from "@/platform/view";

export const TAB: Panel.Tab = {
  Icon: Ranger.ExplorerIcon,
  Name: Panel.createStaticTabName({
    name: "Range explorer",
    icon: <Ranger.ExplorerIcon />,
  }),
  Content: Panel.lazyComponent(
    () => import("@/feature/range/explorer/Explorer"),
    "Explorer",
  ),
};

export const TAB_TYPE = "range_explorer";

export const TABS: Panel.Tabs = { [TAB_TYPE]: TAB };

export const useOpenTab = (): (() => void) =>
  View.useOpenExplorer({ resourceType: "range", tabType: TAB_TYPE });
