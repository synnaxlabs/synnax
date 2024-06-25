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
import { Synnax as PSynnax, useDebouncedCallback } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useStore } from "react-redux";

export type UseSyncerArgs<S, T> = {
  client: Synnax;
  action: T;
  store: Store<S, PayloadAction<T>>;
};

export type Syncer<S, T> = (args: UseSyncerArgs<S, T>) => void;

export const useSyncerDispatch = <S extends {}, T>(
  f: Syncer<S, T>,
  debounce: number = 0,
): Dispatch<PayloadAction<T>> => {
  const client = PSynnax.use();
  const store = useStore<S, PayloadAction<T>>();
  const update = useDebouncedCallback(
    (r: PayloadAction<T>) => {
      if (client != null) f({ client, action: r.payload, store });
      return r;
    },
    debounce,
    [f, store, client],
  );
  return useCallback(
    (a) => {
      const r = store.dispatch(a);
      update(r);
      return r;
    },
    [update],
  );
};
