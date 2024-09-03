// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type PayloadAction, type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { useEffect } from "react";
import { useStore } from "react-redux";

import { useDispatchEffect } from "@/hooks/dispatchers";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { selectActiveKey, useSelectActiveKey } from "@/workspace/selectors";

// this fixes conflicts between JSX and TS
// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-constraint
export const useSyncComponent = <P extends unknown>(
  name: string,
  layoutKey: string,
  save: (workspace: string, store: Store<RootState>, client: Synnax) => Promise<void>,
): Dispatch<PayloadAction<P>> => {
  const client = PSynnax.use();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();
  const syncLayout = useMutation<void, Error>({
    retry: 3,
    mutationKey: [],
    mutationFn: async () => {
      if (layoutKey == null || client == null) return;
      const ws = selectActiveKey(store.getState());
      if (ws == null) return;
      await save(ws, store, client);
    },
    onError: (e) => {
      let message = `Failed to save layout ${name}`;
      if (layoutKey != null) {
        const data = Layout.select(store.getState(), layoutKey);
        if (data != null) message = `Failed to save ${data.name}`;
      }
      addStatus({
        variant: "error",
        message,
        description: e.message,
      });
    },
  });
  const ws = useSelectActiveKey();
  useEffect(() => {
    if (ws == null) return;
    syncLayout.mutate();
  }, [ws]);
  return useDispatchEffect<P>(syncLayout.mutate, 1000);
};
