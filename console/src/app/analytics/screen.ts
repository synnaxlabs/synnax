// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, query, type Synnax as Client } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import { useEffect } from "react";

import { Analytics } from "@/platform/analytics";
import { Session } from "@/session";

interface WatchParams {
  store: Session.Store;
  client: Client;
  screen: Analytics.Sink["screen"];
}

/**
 * Reports the tab the user looks at as a screen until the returned destructor is
 * called. The focused tab comes from the store and its type from the panel the mosaic
 * has already retrieved, so nothing here renders or fetches.
 */
const watch = ({ store, client, screen }: WatchParams): destructor.Destructor => {
  let key: panel.Key | undefined;
  let tabKey: panel.TabKey | undefined;
  let reported: string | undefined;
  let unwatchPanel: destructor.Destructor | undefined;
  const report = (): void => {
    if (key == null || tabKey == null) return;
    const cached = client.panels.getCached(key);
    if (cached == null || !query.isLive(cached)) return;
    const tab = panel.findTab(cached.root, tabKey);
    if (tab == null) return;
    const type = tab.variant === "resource" ? tab.resource.type : tab.type;
    if (type === reported) return;
    reported = type;
    screen(type);
  };
  const follow = (): void => {
    const state = store.getState();
    const nextKey = Session.Panel.selectSelected(state);
    const nextTabKey = Session.Panel.selectSelectedTabs(state)[0];
    if (nextKey === key && nextTabKey === tabKey) return;
    if (nextKey !== key) {
      unwatchPanel?.();
      // The mosaic retrieves the panel. This only follows what lands in the cache, so
      // the first screen of a launch waits for that retrieve instead of racing it.
      unwatchPanel =
        nextKey == null ? undefined : client.panels.onChange(nextKey, report);
    }
    key = nextKey;
    tabKey = nextTabKey;
    report();
  };
  const unwatchStore = store.subscribe(follow);
  follow();
  return () => {
    unwatchStore();
    unwatchPanel?.();
  };
};

/** Mounts the screen watcher for this window. */
export const Screen = (): null => {
  const { screen } = Analytics.use();
  const store = Session.useStore();
  const client = Synnax.use();
  useEffect(() => {
    if (client == null) return;
    return watch({ store, client, screen });
  }, [store, client, screen]);
  return null;
};
