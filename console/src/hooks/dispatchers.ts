// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { type Store, type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { Synnax as PSynnax, useDebouncedCallback } from "@synnaxlabs/pluto";
import { useStore } from "react-redux";

export type Syncer<S, T> = (
  s: Synnax,
  a: T,
  store: Store<S, PayloadAction<T>>
) => Promise<void>;

export const useSyncerDispatch = <S, T>(
  f: Syncer<S, T>,
  debounce: number = 0
): Dispatch<PayloadAction<T>> => {
  const client = PSynnax.use();
  const store = useStore<S, PayloadAction<T>>();
  const update = useDebouncedCallback(
    (r: PayloadAction<T>) => {
      if (client != null) void f(client, r.payload, store);
      return r;
    },
    debounce,
    [f, store, client]
  );

  return useCallback(
    (a) => {
      const r = store.dispatch(a);
      update(r);
      return r;
    },
    [update]
  );
};
