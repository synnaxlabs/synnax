// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { QueryError, type Synnax } from "@synnaxlabs/client";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type Syncer } from "@/hooks/dispatchers";
import { Layout } from "@/layout";
import { selectActiveKey } from "@/workspace/selectors";
import { setActive, type StoreState } from "@/workspace/slice";

export const useLayoutSyncer = (): Syncer<Layout.StoreState & StoreState, any> => {
  return useCallback(async (client, _, store) => {
    if (client == null) return;
    void syncLayout(store, client);
  }, []);
};

export const syncLayout = async (
  store: Store<Layout.StoreState & StoreState>,
  client: Synnax,
): Promise<void> => {
  const s = store.getState();
  const key = selectActiveKey(s);
  if (key == null) return;
  const layoutSlice = Layout.selectSliceState(s);
  try {
    await client.workspaces.setLayout(key, layoutSlice as unknown as UnknownRecord);
  } catch (e) {
    if (e instanceof QueryError) store.dispatch(setActive(null));
    throw e;
  }
};
