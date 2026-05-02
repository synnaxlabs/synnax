// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { useCallback, useRef } from "react";
import { useSyncExternalStoreWithSelector } from "use-sync-external-store/with-selector";

import { type base } from "@/flux/base";
import { useStore } from "@/flux/Provider";
import { useMemoDeepEqual } from "@/memo";

export interface CreateSelectorParams<
  ScopedStore extends base.Store,
  Args extends {},
  Selected,
> {
  subscribe: (
    store: ScopedStore,
    args: Args,
    notify: () => void,
  ) => destructor.Destructor;
  select: (store: ScopedStore, args: Args) => Selected;
  equal?: (a: Selected, b: Selected) => boolean;
}

export type UseSelect<Args extends {}, Selected> = (args: Args) => Selected;

export const createSelector =
  <ScopedStore extends base.Store, Args extends {}, Selected>(
    params: CreateSelectorParams<ScopedStore, Args, Selected>,
  ): UseSelect<Args, Selected> =>
  (args: Args): Selected => {
    const store = useStore<ScopedStore>();
    const memoArgs = useMemoDeepEqual(args);
    const versionRef = useRef(0);

    const subscribe = useCallback(
      (onStoreChange: () => void) =>
        params.subscribe(store, memoArgs, () => {
          versionRef.current++;
          onStoreChange();
        }),
      [store, memoArgs],
    );

    const getSnapshot = useCallback(() => versionRef.current, []);
    const selector = useCallback(
      () => params.select(store, memoArgs),
      [store, memoArgs],
    );

    return useSyncExternalStoreWithSelector(
      subscribe,
      getSnapshot,
      undefined,
      selector,
      params.equal,
    );
  };
