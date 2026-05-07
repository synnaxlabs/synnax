// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";
import { use, useCallback, useSyncExternalStore } from "react";

import { type base } from "@/flux/base";
import { hashQuery } from "@/flux/base/queryCache";
import { useQueryCache, useStore } from "@/flux/Provider";
import { pendingResult, successResult } from "@/flux/result";
import { useMemoDeepEqual } from "@/memo";
import { state } from "@/state";
import { Synnax } from "@/synnax";

export interface RetrieveParams<
  Query extends base.Shape,
  Store extends base.Store,
  AllowDisconnected extends boolean = false,
> {
  client: AllowDisconnected extends true ? Client | null : Client;
  query: Query;
  store: Store;
}

export interface MountListenersParams<
  Query extends base.Shape,
  Data extends base.Shape,
  Store extends base.Store,
  AllowDisconnected extends boolean = false,
> extends RetrieveParams<Query, Store, AllowDisconnected> {
  /// Push an updated value into the cache. Accepts either the next value
  /// directly or a setter `(prev) => next` that computes the next value from
  /// the cached value. A setter that returns `undefined` is a no-op, which lets
  /// listeners short-circuit when there is no current value to merge against.
  onChange: (value: state.SetArg<Data | undefined>) => void;
}

export interface CreateRetrieveParams<
  Query extends base.Shape,
  Data extends base.Shape,
  Store extends base.Store,
  AllowDisconnected extends boolean = false,
> {
  name: string;
  retrieve: (params: RetrieveParams<Query, Store, AllowDisconnected>) => Promise<Data>;
  mountListeners?: (
    params: MountListenersParams<Query, Data, Store, AllowDisconnected>,
  ) => destructor.Destructor | destructor.Destructor[];
  allowDisconnected?: AllowDisconnected;
}

export type UseRetrieve<Query extends base.Shape, Data extends base.Shape> = (
  query: Query,
) => Data;

/// Builds a Suspense-shaped retrieve hook. The returned hook reads the
/// per-client query cache and either returns the resolved value or suspends on
/// the in-flight promise. Concurrent reads for the same query share one fetch
/// via the cache. Channel-listener-driven updates push new values into the
/// cache, which notifies subscribed consumers without re-suspending.
export const createRetrieve = <
  Query extends base.Shape,
  Data extends base.Shape,
  ScopedStore extends base.Store = {},
  AllowDisconnected extends boolean = false,
>(
  createParams: CreateRetrieveParams<Query, Data, ScopedStore, AllowDisconnected>,
): UseRetrieve<Query, Data> => {
  const { name, retrieve, mountListeners, allowDisconnected } = createParams;
  return (query: Query): Data => {
    const memoQuery = useMemoDeepEqual(query);
    const client = Synnax.use();
    const store = useStore<ScopedStore>();
    const cache = useQueryCache();
    const hash = hashQuery({ name, query: memoQuery });

    if (client == null && !allowDisconnected)
      throw new Error(
        `Cannot retrieve ${name}: no Synnax client connected. Pass allowDisconnected to opt out.`,
      );

    useSyncExternalStore(
      useCallback(
        (notify) => {
          const cacheSub = cache.subscribe(hash, notify);
          if (mountListeners == null || client == null) return cacheSub;
          const onChange = (value: state.SetArg<Data | undefined>) => {
            const current = cache.get<Data>(hash);
            const prev = current?.variant === "success" ? current.data : undefined;
            const next = state.executeSetter(value, prev);
            if (next == null) return;
            cache.set(hash, successResult(name, next));
          };
          const result = mountListeners({
            client: client as AllowDisconnected extends true ? Client | null : Client,
            store,
            query: memoQuery,
            onChange,
          });
          const listeners = Array.isArray(result) ? result : [result];
          return () => {
            cacheSub();
            listeners.forEach((d) => d?.());
          };
        },
        [cache, hash, client, store, memoQuery],
      ),
      useCallback(() => cache.get(hash), [cache, hash]),
    );

    const entry = cache.get<Data>(hash);
    if (entry?.variant === "success") return entry.data;
    if (entry?.variant === "error")
      // Throwing the Status: Flux.Boundary normalizes thrown Statuses without
      // re-running fromException, preserving the resource-specific message.
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw entry.status;
    if (entry?.variant === "loading" && entry.promise != null)
      return use(entry.promise);

    const promise = retrieve({
      client: client as AllowDisconnected extends true ? Client | null : Client,
      query: memoQuery,
      store,
    });
    cache.set(hash, pendingResult(name, promise));
    return use(promise);
  };
};
