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
import { Flux, type Pluto } from "@synnaxlabs/pluto";
import { useCallback, useEffect } from "react";
import { useStore } from "react-redux";

import { useDispatchEffect } from "@/hooks/useDispatchEffect";
import { type RootState } from "@/store";
import { selectActiveKey, useSelectActiveKey } from "@/workspace/selectors";

interface UpdateParams {
  store: Store<RootState>;
  layoutKey: string;
}

export interface SaveArgs {
  key: string;
  workspace: string;
  store: Store<RootState>;
  fluxStore: Pluto.FluxStore;
  client: Client;
}

export const createSyncComponent = (
  name: string,
  save: (args: SaveArgs) => Promise<void>,
) => {
  const { useUpdate } = Flux.createUpdate<UpdateParams, Pluto.FluxStore>({
    name,
    verbs: Flux.SAVE_VERBS,
    update: async ({ client, data, store: fluxStore }) => {
      const { store, layoutKey } = data;
      if (layoutKey == null || client == null) return false;
      const ws = selectActiveKey(store.getState());
      if (ws == null) return false;
      await save({ key: layoutKey, workspace: ws, store, fluxStore, client });
      return data;
    },
  });
  return <P>(
    layoutKey: string,
    dispatch?: Dispatch<PayloadAction<P>>,
  ): Dispatch<PayloadAction<P>> => {
    const { update } = useUpdate();
    const store = useStore<RootState>();
    const run = useCallback(() => {
      update({ layoutKey, store });
    }, [layoutKey, store]);
    const ws = useSelectActiveKey();
    useEffect(() => run(), [ws, run]);
    return useDispatchEffect(run, 100, dispatch);
  };
};
