// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type PayloadAction, type Store } from "@reduxjs/toolkit";
import { type Synnax as Client } from "@synnaxlabs/client";
import { Flux, Synnax } from "@synnaxlabs/pluto";
import { useCallback, useEffect } from "react";
import { useStore } from "react-redux";

import { useDispatchEffect } from "@/hooks/useDispatchEffect";
import { type RootState } from "@/store";
import { selectActiveKey, useSelectActiveKey } from "@/workspace/selectors";

export const useSyncComponent = <P>(
  name: string,
  layoutKey: string,
  save: (workspace: string, store: Store<RootState>, client: Client) => Promise<void>,
  dispatch?: Dispatch<PayloadAction<P>>,
): Dispatch<PayloadAction<P>> => {
  const client = Synnax.use();
  const store = useStore<RootState>();
  const syncLayout = Flux.useAction({
    resourceName: name,
    opName: "Save",
    action: useCallback(async () => {
      if (layoutKey == null || client == null) return;
      const ws = selectActiveKey(store.getState());
      if (ws == null) return;
      await save(ws, store, client);
    }, [layoutKey, client, store, save]),
  });
  const ws = useSelectActiveKey();
  useEffect(() => {
    if (ws == null) return;
    syncLayout.run();
  }, [ws]);
  return useDispatchEffect<P>(syncLayout.run, 1000, dispatch);
};
