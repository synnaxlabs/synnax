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
  Raw = Selected,
> {
  subscribe: (
    store: ScopedStore,
    args: Args,
    notify: () => void,
  ) => destructor.Destructor;
  select: (store: ScopedStore, args: Args) => Raw;
  /**
   * transform derives the selected value from the raw selection. It is
   * memoized on the raw value's reference: as long as select returns the same
   * reference (e.g. a stored array that only changes when its contents change),
   * transform is not re-run and the previous result is returned, so a derived
   * mapping does not cause re-renders on unrelated store updates.
   */
  transform?: (raw: Raw, args: Args) => Selected;
  equal?: (a: Selected, b: Selected) => boolean;
}

export type UseSelect<Args extends {}, Selected> = (args: Args) => Selected;

export const createSelector =
  <ScopedStore extends base.Store, Args extends {}, Selected, Raw = Selected>(
    params: CreateSelectorParams<ScopedStore, Args, Selected, Raw>,
  ): UseSelect<Args, Selected> =>
  (args: Args): Selected => {
    const store = useStore<ScopedStore>();
    const memoArgs = useMemoDeepEqual(args);
    const versionRef = useRef(0);
    const cacheRef = useRef<{ raw: Raw; out: Selected } | null>(null);

    const subscribe = useCallback(
      (onStoreChange: () => void) =>
        params.subscribe(store, memoArgs, () => {
          versionRef.current++;
          onStoreChange();
        }),
      [store, memoArgs],
    );

    const getSnapshot = useCallback(() => versionRef.current, []);
    const selector = useCallback((): Selected => {
      const raw = params.select(store, memoArgs);
      if (params.transform == null) return raw as unknown as Selected;
      const cache = cacheRef.current;
      if (cache != null && cache.raw === raw) return cache.out;
      const out = params.transform(raw, memoArgs);
      cacheRef.current = { raw, out };
      return out;
    }, [store, memoArgs]);

    return useSyncExternalStoreWithSelector(
      subscribe,
      getSnapshot,
      undefined,
      selector,
      params.equal,
    );
  };
