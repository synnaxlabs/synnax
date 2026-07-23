// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type ranger } from "@synnaxlabs/client";
import { Ranger, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { remove, updateRemote } from "@/session/range/slice";
// Type-only: a value import of the store would cycle through its domain barrels.
import type { Action, State } from "@/session/store";
import { Synchronizer } from "@/session/synchronizer";

const useSyncRanges = (): void => {
  const dispatch = useDispatch<Dispatch<Action>>();
  const store = useStore<State>();
  const client = Synnax.use();
  const handleRangeSet = useCallback(
    ({ timeRange, ...rest }: ranger.Payload): void => {
      dispatch(
        updateRemote({ ...rest, parent: undefined, timeRange: timeRange.numeric }),
      );
    },
    [dispatch],
  );
  Ranger.useSetSynchronizer(handleRangeSet);
  const handleRangeDelete = useCallback(
    (key: ranger.Key) => {
      dispatch(remove({ keys: [key] }));
    },
    [dispatch],
  );
  Ranger.useDeleteSynchronizer(handleRangeDelete);
  Synchronizer.useEpochSweep(async () => {
    if (client == null) return;
    // Unpersisted ranges are local-only; only cluster-backed ones can vanish.
    const keys = store
      .getState()
      .range.ranges.filter(({ persisted }) => persisted)
      .map(({ key }) => key);
    if (keys.length === 0) return;
    const found = await client.ranges.retrieve(keys);
    const foundKeys = new Set(found.map(({ key }) => key));
    const missing = keys.filter((key) => !foundKeys.has(key));
    if (missing.length > 0) dispatch(remove({ keys: missing }));
  });
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers = { useSyncRanges };
