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
import { remove, type StoreState } from "@/session/core/slice";
import { Persist } from "@/session/persist";

/**
 * Purges the state of every cluster the removed Cores were the last to name. Reads
 * before the removal lands, since the reducer drops the records it needs.
 */
const purgeOnRemove: Middleware<record.Unknown> = (store) => (next) => (action) => {
  if (!remove.match(action)) return next(action);
  const state = store.getState() as StoreState;
  const removed = array.toArray(action.payload);
  const orphaned = unique
    .unique(removed.flatMap((key) => selectState(state, key)?.clusterKey ?? []))
    .filter((cluster) => selectIsClusterOrphaned(state, cluster, removed));
  const result = next(action);
  orphaned.forEach((cluster) => store.dispatch(Persist.purge(cluster)));
  return result;
};

export const MIDDLEWARE = [purgeOnRemove];
