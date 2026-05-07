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
  /// Subscribe to store mutations that may change the selected value. The
  /// callback fires on every relevant change. The selector retains its current
  /// value if `equal` reports no change.
  subscribe: (
    store: ScopedStore,
    args: Args,
    notify: () => void,
  ) => destructor.Destructor;
  /// Compute the selected slice from the current store state. Called only
  /// after the parent record (typically loaded by a sibling `useRetrieve`) is
  /// hydrated. Selectors are no longer required to throw `NotFoundError` on
  /// missing parents - they return whatever the slice resolves to (which may
  /// itself be undefined for missing sub-keys).
  select: (store: ScopedStore, args: Args) => Selected;
  equal?: (a: Selected, b: Selected) => boolean;
}

export type UseSelect<Args extends {}, Selected> = (args: Args) => Selected;

/// Suspending selector. The returned hook reads a slice of the per-record
/// store and stays in sync with mutations via `useSyncExternalStoreWithSelector`.
/// It does not retrieve - the caller is responsible for ensuring the parent
/// record is hydrated, typically by calling `useRetrieve` in a sibling or
/// ancestor component. If the parent record is absent when the selector runs,
/// the selector returns whatever its `select` function resolves to (often
/// `undefined`); it does not throw. Suspension on a missing parent record is
/// the responsibility of the corresponding `useRetrieve`, not the selector.
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
