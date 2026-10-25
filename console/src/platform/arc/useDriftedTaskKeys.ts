// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, query, task } from "@synnaxlabs/client";
import { Store } from "@synnaxlabs/lyra/store";
import { Synnax } from "@synnaxlabs/pluto";
import { destructor } from "@synnaxlabs/x";

/**
 * Returns the keys of the cached tasks of the given Arcs that drifted from their
 * deployment, and re-renders when any of those tasks changes.
 */
export const useDriftedTaskKeys = (keys: arc.Key[]): task.Key[] => {
  const client = Synnax.use();
  const tasks = Store.useItems({
    keys,
    subscribe: (listener, key) =>
      client?.arcs.task.onChange(key, listener) ?? destructor.NOOP,
    getItem: (arcKeys) =>
      arcKeys
        .map((key) => client?.arcs.task.getCached(key))
        .filter((tsk): tsk is task.Task => query.isLive(tsk) && tsk != null),
  });
  return tasks.filter((tsk) => task.drifted(tsk)).map(({ key }) => key);
};
