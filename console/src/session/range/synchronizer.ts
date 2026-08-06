// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor } from "@synnaxlabs/x";

import {
  type Action,
  remove,
  type StoreState,
  updateRemote,
} from "@/session/range/slice";
import { type Synchronizer } from "@/session/synchronizer";

const syncRanges: Synchronizer.Callbacks<StoreState, Action> = {
  reconcile: async ({ client, store }) => {
    // Unpersisted ranges are local-only; only cluster-backed ones can vanish.
    const keys = store
      .getState()
      .range.ranges.filter(({ persisted }) => persisted)
      .map(({ key }) => key);
    if (keys.length === 0) return;
    const found = await client.ranges.retrieve({ keys, ignoreNotFoundError: true });
    const foundKeys = new Set(found.map(({ key }) => key));
    const missing = keys.filter((key) => !foundKeys.has(key));
    if (missing.length > 0) store.dispatch(remove({ keys: missing }));
  },
  listen: ({ client, store }) => {
    const removeOnSet = client.ranges.onSet(({ payload: { timeRange, ...rest } }) => {
      store.dispatch(
        updateRemote({ ...rest, parent: undefined, timeRange: timeRange.numeric }),
      );
    });
    const removeOnDelete = client.ranges.onDelete((key) =>
      store.dispatch(remove({ keys: [key] })),
    );
    return () => destructor.unwind(removeOnSet, removeOnDelete);
  },
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<StoreState, Action> = [
  { name: "sync ranges", use: () => syncRanges },
];
