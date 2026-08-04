// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type destructor, type optional } from "@synnaxlabs/x";
import { useCallback, useRef } from "react";
import { useSyncExternalStoreWithSelector } from "use-sync-external-store/with-selector";

import { useMemoDeepEqual } from "@/memo";
import { Synnax } from "@/synnax";

/** Client and arguments handed to a selector's subscribe and select members. */
export interface SelectorParams<Args extends {}> {
  /** Null when no cluster is connected; selectors treat it as an empty cache. */
  client: Client | null;
  args: Args;
}

export interface CreateSelectorParams<Args extends {}, Selected, Raw = Selected> {
  subscribe: (
    params: SelectorParams<Args>,
    notify: () => void,
  ) => destructor.Destructor;
  select: (params: SelectorParams<Args>) => Raw;
  /**
   * transform derives the selected value from the raw selection. It is memoized
   * on the raw value's reference and the args: as long as select returns the same
   * reference (e.g. a cached array that only changes when its contents change) and
   * the args are unchanged, transform is not re-run and the previous result is
   * returned, so a derived mapping does not cause re-renders on unrelated cache
   * updates. It does re-run when an arg it depends on changes, even if the raw
   * selection is referentially stable.
   */
  transform?: (raw: Raw, args: Args) => Selected;
  equal?: (a: Selected, b: Selected) => boolean;
}

/** Getter reads the instantaneous selected value from the cache when called. */
export type Getter<Args extends {}, Selected> = optional.Arg<Args, Selected>;

/** UseSelect is the reactive hook: it subscribes and re-renders on change. */
export type UseSelect<Args extends {}, Selected> = optional.Arg<Args, Selected>;

/**
 * UseGet is the non-reactive hook: it returns a stable {@link Getter} that reads the
 * current value from the cache on demand without subscribing. Use it to read a value
 * inside a callback without mounting a reactive subscription.
 */
export type UseGet<Args extends {}, Selected> = () => Getter<Args, Selected>;

/**
 * Selector is the pair returned by {@link createSelector}: the reactive hook first, the
 * non-reactive getter hook second. Destructure the pair to name each.
 */
export type Selector<Args extends {}, Selected> = [
  UseSelect<Args, Selected>,
  UseGet<Args, Selected>,
];

const computeSelected = <Args extends {}, Selected, Raw>(
  params: CreateSelectorParams<Args, Selected, Raw>,
  client: Client | null,
  args: Args,
): Selected => {
  const raw = params.select({ client, args });
  if (params.transform == null) return raw as unknown as Selected;
  return params.transform(raw, args);
};

export const createSelector = <Args extends {}, Selected, Raw = Selected>(
  params: CreateSelectorParams<Args, Selected, Raw>,
): Selector<Args, Selected> => {
  const useSelect = (args: Args = {} as Args): Selected => {
    const client = Synnax.use();
    const memoArgs = useMemoDeepEqual(args);
    const versionRef = useRef(0);
    const cacheRef = useRef<{ raw: Raw; args: Args; out: Selected } | null>(null);

    const subscribe = useCallback(
      (onStoreChange: () => void) =>
        params.subscribe({ client, args: memoArgs }, () => {
          versionRef.current++;
          onStoreChange();
        }),
      [client, memoArgs],
    );

    const getSnapshot = useCallback(() => versionRef.current, []);
    const selector = useCallback((): Selected => {
      const raw = params.select({ client, args: memoArgs });
      if (params.transform == null) return raw as unknown as Selected;
      const cache = cacheRef.current;
      if (cache != null && cache.raw === raw && cache.args === memoArgs)
        return cache.out;
      const out = params.transform(raw, memoArgs);
      cacheRef.current = { raw, args: memoArgs, out };
      return out;
    }, [client, memoArgs]);

    return useSyncExternalStoreWithSelector(
      subscribe,
      getSnapshot,
      undefined,
      selector,
      params.equal,
    );
  };

  const useGet = (): Getter<Args, Selected> => {
    const client = Synnax.use();
    return useCallback(
      (args: Args = {} as Args) => computeSelected(params, client, args),
      [client],
    );
  };

  return [useSelect, useGet];
};
