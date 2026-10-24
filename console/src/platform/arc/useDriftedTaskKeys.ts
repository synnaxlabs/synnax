// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, query, task } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { destructor } from "@synnaxlabs/x";

import { ExternalStore } from "@/platform/external-store";

/**
 * Returns the keys of the cached tasks of the given Arcs that drifted from their
 * deployment, and re-renders when any of those tasks changes.
 */
export const useDriftedTaskKeys = (keys: arc.Key[]): task.Key[] => {
  const client = Synnax.use();
  return ExternalStore.useItems({
    keys,
    subscribe: (handler, key) =>
      client?.arcs.task.onChange(key, handler) ?? destructor.NOOP,
    getItems: (arcKeys) =>
      arcKeys
        .map((key) => client?.arcs.task.getCached(key))
        .filter(
          (tsk): tsk is task.Task =>
            query.isLive(tsk) && tsk != null && task.drifted(tsk),
        )
        .map(({ key }) => key),
  });
};
