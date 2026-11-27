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
import { Session } from "@/session";

const TAB_TYPE = "settings";

const TAB = {
  Icon: Icon.Settings,
  iconKey: "gear",
  Name: Panel.createStaticTabName({ name: "Settings", icon: <Icon.Settings /> }),
  Content: Panel.lazyComponent(() => import("@/app/settings/Settings"), "Settings"),
} satisfies Panel.Tab;

export const TABS = { [TAB_TYPE]: TAB } satisfies Panel.Tabs;

/** The key of the License section. */
export const LICENSE_SECTION = "license";

/**
 * Returns a function that opens the Settings tab in a draft of the user, or selects it
 * when a draft holds it. The tab shows the given section, else the section the window
 * showed last.
 */
export const useOpen = (section?: string): (() => void) => {
  const openTab = Panel.useOpenTab();
  const dispatch = Session.useDispatch();
  return () => {
    if (section != null)
      dispatch(Session.Nav.selectSection({ page: "settings", section }));
    openTab(
      { variant: "view", type: TAB_TYPE },
      { singleton: true, draft: { name: "Settings" } },
    );
  };
};
