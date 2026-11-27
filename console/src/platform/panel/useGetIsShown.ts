// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, panel, type resource } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Synnax } from "@synnaxlabs/pluto";
import { useStore } from "react-redux";

import { Session } from "@/session";

/**
 * @returns a function that resolves whether a focused window shows a tab of the
 * resource. It retrieves the selected panel of each focused window from the Core.
 * @throws DisconnectedError when no Core is connected.
 */
export const useGetIsShown = (): ((id: resource.ID) => Promise<boolean>) => {
  const store = useStore<Session.State>();
  const client = Synnax.use();
  return async (id) => {
    if (client == null) throw new DisconnectedError();
    const state = store.getState();
    const current = Drift.selectWindowKey(state);
    for (const { key, focus } of Drift.selectWindows(state)) {
      // Drift tracks focus only on the desktop runtime, so this window asks the DOM.
      const focused = key === current ? document.hasFocus() : focus === true;
      const visible = focused ? Session.Panel.selectVisibleTabs(state, key) : null;
      if (visible == null) continue;
      const { root } = await client.panels.retrieve(visible.panel);
      const tab = panel.findTabByResource(root, id);
      if (tab != null && visible.tabs.includes(tab.key)) return true;
    }
    return false;
  };
};
