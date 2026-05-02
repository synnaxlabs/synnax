// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { useCallback, useRef, useSyncExternalStore } from "react";

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

export const createSelector = <
  ScopedStore extends base.Store,
  Args extends {},
  Selected,
>(
  params: CreateSelectorParams<ScopedStore, Args, Selected>,
): UseSelect<Args, Selected> => {
  const equal = params.equal ?? Object.is;

  return (args: Args): Selected => {
    const store = useStore<ScopedStore>();
    const memoArgs = useMemoDeepEqual(args);

    // version is bumped by subscribe's notify wrapper. getSnapshot only
    // re-evaluates select when the version changes, preventing infinite
    // re-render loops when select returns new object references from
    // unchanged store data.
    const cache = useRef<{
      version: number;
      args: Args;
      selected: Selected;
    } | null>(null);
    const version = useRef(0);

    const subscribe = useCallback(
      (onStoreChange: () => void) =>
        params.subscribe(store, memoArgs, () => {
          version.current++;
          onStoreChange();
        }),
      [store, memoArgs],
    );

    const getSnapshot = useCallback((): Selected => {
      const prev = cache.current;
      if (prev !== null && prev.version === version.current && prev.args === memoArgs)
        return prev.selected;
      const nextSelected = params.select(store, memoArgs);
      if (prev !== null && equal(nextSelected, prev.selected)) {
        cache.current = {
          version: version.current,
          args: memoArgs,
          selected: prev.selected,
        };
        return prev.selected;
      }
      cache.current = {
        version: version.current,
        args: memoArgs,
        selected: nextSelected,
      };
      return nextSelected;
    }, [store, memoArgs]);

    return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
  };
};
