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

const TAB_TYPE = "library";

const TAB = {
  Icon: Icon.Library,
  iconKey: "library",
  Name: Panel.createStaticTabName({ name: "Library", icon: <Icon.Library /> }),
  Content: Panel.lazyComponent(() => import("@/app/library/Library"), "Library"),
} satisfies Panel.Tab;

export const TABS = { [TAB_TYPE]: TAB } satisfies Panel.Tabs;

/** The key of the Labels section. */
export const LABELS_SECTION = "labels";

/**
 * Returns a function that opens the Library tab in a draft of the user, or selects it
 * when a draft holds it. The tab shows the given section, else the section the window
 * showed last.
 */
export const useOpen = (section?: string): (() => void) => {
  const openTab = Panel.useOpenTab();
  const dispatch = Session.useDispatch();
  return () => {
    if (section != null)
      dispatch(Session.Nav.selectSection({ page: "library", section }));
    openTab(
      { variant: "view", type: TAB_TYPE },
      { singleton: true, draft: { name: "Library" } },
    );
  };
};
