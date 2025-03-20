// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { ChannelServices } from "@/channel/services";
import { Hardware } from "@/hardware";
import { Device } from "@/hardware/device";
import { Layout } from "@/layout";
import { Range } from "@/range";
import { UserServices } from "@/user/services";
import { Vis } from "@/vis";
import { WorkspaceServices } from "@/workspace/services";

export const NAV_DRAWER_ITEMS: Layout.NavDrawerItem[] = [
  ...Hardware.NAV_DRAWER_ITEMS,
  Range.TOOLBAR,
  Vis.TOOLBAR,
  Device.TOOLBAR,
  ChannelServices.TOOLBAR,
  WorkspaceServices.TOOLBAR,
  UserServices.TOOLBAR,
];

const MODE_CONFIG: Triggers.ModeConfig<string> = Object.fromEntries(
  NAV_DRAWER_ITEMS.filter((item) => item.trigger?.length > 0).flatMap((item) => [
    [item.key, [item.trigger]],
    [`${item.key}-double`, [[item.trigger[0], item.trigger[0]]]],
  ]),
) as Triggers.ModeConfig<string>;

const flattenedConfig = Triggers.flattenConfig(MODE_CONFIG);

export const useTriggers = () => {
  const dispatch = useDispatch();
  Triggers.use({
    triggers: flattenedConfig,
    callback: (e) => {
      console.log(e);
      if (e.stage === "end") return;
      const mode = Triggers.determineMode(MODE_CONFIG, e.triggers);
      if (mode.length === 0) return;
      if (mode.includes("double")) {
        const key = mode.split("-")[0];
        dispatch(Layout.setNavDrawerVisible({ windowKey: "main", key, value: true }));
      } else dispatch(Layout.toggleNavHover({ windowKey: "main", key: mode }));
    },
  });
};
