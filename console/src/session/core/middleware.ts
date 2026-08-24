// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware } from "@reduxjs/toolkit";
import { array, type record, unique } from "@synnaxlabs/x";

import { selectIsClusterOrphaned, selectState } from "@/session/core/selectors";
import { remove, set, type StoreState } from "@/session/core/slice";
import { Persist } from "@/session/persist";

/**
 * Purges the state of every cluster the change left unreachable. A Core stops naming
 * its cluster when it is removed and when it is repointed at another address, and the
 * cluster's stored state has nothing to open it once no Core names it.
 */
const purgeOrphanedClusters: Middleware<record.Unknown> =
  (store) => (next) => (action) => {
    const isRemove = remove.match(action);
    if (!isRemove && !set.match(action)) return next(action);
    const state = store.getState() as StoreState;
    const keys = isRemove ? array.toArray(action.payload) : [action.payload.key];
    // Read the cached clusters first: the reducer is about to drop them.
    const named = unique.unique(
      keys.flatMap((key) => selectState(state, key)?.clusterKey ?? []),
    );
    const result = next(action);
    const settled = store.getState() as StoreState;
    named
      .filter((cluster) => selectIsClusterOrphaned(settled, cluster))
      .forEach((cluster) => store.dispatch(Persist.purge(cluster)));
    return result;
  };

export const MIDDLEWARE = [purgeOrphanedClusters];
