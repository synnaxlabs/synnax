// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, query, type Synnax as Client, task } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import { useMemo, useSyncExternalStore } from "react";

const NO_KEYS: task.Key[] = [];

const createStore = (client: Client | null, keys: arc.Key[]) => {
  let snapshot = NO_KEYS;
  return {
    subscribe: (handler: () => void): destructor.Destructor => {
      if (client == null) return () => {};
      const releases = keys.map((key) => client.arcs.task.onChange(key, handler));
      return () => releases.forEach((release) => release());
    },
    getSnapshot: (): task.Key[] => {
      if (client == null) return snapshot;
      const next = keys
        .map((key) => client.arcs.task.getCached(key))
        .filter(
          (tsk): tsk is task.Task =>
            query.isLive(tsk) && tsk != null && task.drifted(tsk),
        )
        .map(({ key }) => key);
      if (next.length !== snapshot.length || next.some((k, i) => k !== snapshot[i]))
        snapshot = next;
      return snapshot;
    },
  };
};

/**
 * Returns the keys of the cached tasks of the given Arcs that drifted from their
 * deployment, and re-renders when any of those tasks changes.
 */
export const useDriftedTaskKeys = (keys: arc.Key[]): task.Key[] => {
  const client = Synnax.useClient();
  const { subscribe, getSnapshot } = useMemo(
    () => createStore(client, keys),
    [client, keys],
  );
  return useSyncExternalStore(subscribe, getSnapshot);
};
