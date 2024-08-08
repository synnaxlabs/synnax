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
import { deep, id, type UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { RootState } from "@/store";
import { selectActiveKey } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

const MAX_RETRY_COUNT = 3;

export const useSyncLayout = async (): Promise<void> => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const prevSync = useRef<unknown>();
  const sync = useMutation({
    mutationKey: ["workspace.save"],
    retry: MAX_RETRY_COUNT,
    mutationFn: useDebouncedCallback(
      async (s: RootState) => {
        const key = selectActiveKey(s);
        if (key == null || client == null) return;
        const layoutSlice = Layout.selectSliceState(s);
        if (deep.equal(prevSync.current, layoutSlice)) return;
        prevSync.current = layoutSlice;
        await client.workspaces.setLayout(key, layoutSlice as unknown as UnknownRecord);
      },
      250,
      [client],
    ),
    onError: (e) => {
      if (QueryError.matches(e)) {
        addStatus({
          key: id.id(),
          variant: "error",
          message: "Layout not found in cluster. Clearing.",
        });
        store.dispatch(setActive(null));
        return;
      }
      addStatus({
        key: id.id(),
        variant: "error",
        message: "Failed to save workspace",
        description: e.message,
      });
    },
  });

  useEffect(() => {
    store.subscribe(() => sync.mutate(store.getState()));
  }, [client]);
};
