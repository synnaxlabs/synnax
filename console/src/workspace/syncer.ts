// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { QueryError } from "@synnaxlabs/client";
import { Status, Synnax, useDebouncedCallback } from "@synnaxlabs/pluto";
import { deep, type UnknownRecord } from "@synnaxlabs/x";
import { useEffect, useReducer, useRef } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { RootState } from "@/store";
import { selectActiveKey } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

export const useSyncLayout = async (): Promise<void> => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const prevSync = useRef<unknown>();
  const sync = useDebouncedCallback(
    (s: RootState): void => {
      const key = selectActiveKey(s);
      if (key == null || client == null) return;
      const layoutSlice = Layout.selectSliceState(s);
      if (deep.equal(prevSync.current, layoutSlice)) return;
      client.workspaces
        .setLayout(key, layoutSlice as unknown as UnknownRecord)
        .catch((e) => {
          if (e instanceof QueryError) {
            addStatus({
              key: "layout-sync",
              variant: "error",
              message: "Layout not found in cluster. Clearing.",
            });
            store.dispatch(setActive(null));
            return;
          }
          addStatus({
            key: "layout-sync",
            variant: "error",
            message: "Failed to sync layout: " + e.message,
          });
        });
    },
    250,
    [client],
  );
  useEffect(() => store.subscribe(() => sync(store.getState())), [client]);
};
