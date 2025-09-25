// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type Destructor } from "@synnaxlabs/x";
import { useCallback, useRef, useState } from "react";

import { type core } from "@/flux/core";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  successResult,
} from "@/flux/result";
import { useAsyncEffect } from "@/hooks";
import { useDestructors } from "@/hooks/useDestructors";
import { useMemoDeepEqual } from "@/memo";
import { state } from "@/state";
import { useAdder } from "@/status/Aggregator";
import { Synnax } from "@/synnax";

export interface RetrieveParams<Query extends core.Shape, Store extends core.Store> {
  client: Client;
  query: Query;
  store: Store;
}

export interface RetrieveMountListenersParams<
  Query extends core.Shape,
  Data extends core.Shape,
  Store extends core.Store,
> extends RetrieveParams<Query, Store> {
  onChange: state.Setter<Data | undefined>;
}

export interface CreateRetrieveParams<
  Query extends core.Shape,
  Data extends core.Shape,
  Store extends core.Store,
> {
  name: string;
  retrieve: (Params: RetrieveParams<Query, Store>) => Promise<Data>;
  mountListeners?: (
    Params: RetrieveMountListenersParams<Query, Data, Store>,
  ) => Destructor | Destructor[];
}

export interface BeforeRetrieveParams<Query extends core.Shape> {
  query: Query;
}

export interface UseObservableBaseRetrieveParams<
  Query extends core.Shape,
  Data extends state.State,
> {
  addStatusOnFailure?: boolean;
  beforeRetrieve?: (Params: BeforeRetrieveParams<Query>) => Data | boolean;
  onChange: (result: state.SetArg<Result<Data>>, query: Query) => void;
  scope?: string;
}

export interface UseRetrieveObservableParams<
  Query extends core.Shape,
  Data extends state.State,
> extends Omit<UseObservableBaseRetrieveParams<Query, Data>, "onChange"> {
  onChange: (result: Result<Data>, query: Query) => void;
}

export interface UseRetrieveObservableReturn<Query extends core.Shape> {
  retrieve: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: core.FetchOptions,
  ) => void;
  retrieveAsync: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: core.FetchOptions,
  ) => Promise<void>;
}

export type UseRetrieveStatefulReturn<
  Query extends core.Shape,
  Data extends state.State,
> = Result<Data> & UseRetrieveObservableReturn<Query>;

export interface UseDirectRetrieveParams<
  Query extends core.Shape,
  Data extends state.State,
> extends Pick<
    UseObservableBaseRetrieveParams<Query, Data>,
    "scope" | "beforeRetrieve" | "addStatusOnFailure"
  > {
  query: Query;
}

export type UseDirectRetrieveReturn<Data extends state.State> = Result<Data>;

export interface UseRetrieveEffectParams<
  Query extends core.Shape,
  Data extends state.State,
> {
  scope?: string;
  onChange?: (result: Result<Data>, query: Query) => void;
  query?: Query;
}

export interface UseRetrieve<Query extends core.Shape, Data extends state.State> {
  (
    params: Query,
    opts?: Pick<
      UseDirectRetrieveParams<Query, Data>,
      "beforeRetrieve" | "addStatusOnFailure"
    >,
  ): UseDirectRetrieveReturn<Data>;
}

export interface UseRetrieveEffect<Query extends core.Shape, Data extends state.State> {
  (params: UseRetrieveEffectParams<Query, Data>): void;
}

export interface UseRetrieveStateful<
  Query extends core.Shape,
  Data extends state.State,
> {
  (): UseRetrieveStatefulReturn<Query, Data>;
}

export interface UseRetrieveObservable<
  Query extends core.Shape,
  Data extends state.State,
> {
  (
    params: UseRetrieveObservableParams<Query, Data>,
  ): UseRetrieveObservableReturn<Query>;
}

export interface CreateRetrieveReturn<
  Query extends core.Shape,
  Data extends state.State,
> {
  useRetrieve: UseRetrieve<Query, Data>;
  useRetrieveEffect: UseRetrieveEffect<Query, Data>;
  useRetrieveStateful: UseRetrieveStateful<Query, Data>;
  useRetrieveObservable: UseRetrieveObservable<Query, Data>;
}

const initialResult = <Data extends state.State>(name: string): Result<Data> =>
  loadingResult<Data>(`Retrieving ${name}`, undefined);

const useStateful = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>(
  createParams: CreateRetrieveParams<Query, Data, ScopedStore>,
): UseRetrieveStatefulReturn<Query, Data> => {
  const [state, setState] = useState<Result<Data>>(
    initialResult<Data>(createParams.name),
  );
  return {
    ...state,
    ...useObservableBase({ ...createParams, onChange: setState }),
  };
};

const useObservableBase = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  retrieve,
  mountListeners,
  name,
  onChange,
  scope,
  beforeRetrieve,
  addStatusOnFailure = true,
}: UseObservableBaseRetrieveParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    ScopedStore
  >): UseRetrieveObservableReturn<Query> => {
  const client = Synnax.use();
  const queryRef = useRef<Query | null>(null);
  const store = useStore<ScopedStore>(scope);
  const listeners = useDestructors();
  const addStatus = useAdder();
  const handleListenerChange = useCallback(
    (value: state.SetArg<Data | undefined>) => {
      if (queryRef.current == null) return;
      onChange((prev) => {
        const next = state.executeSetter(value, prev.data);
        if (next == null) return prev;
        return successResult(`retrieved ${name}`, next);
      }, queryRef.current);
    },
    [onChange, name],
  );
  const retrieveAsync = useCallback(
    async (
      querySetter: state.SetArg<Query, Partial<Query>>,
      options: core.FetchOptions = {},
    ) => {
      const { signal } = options;
      const query = state.executeSetter<Query, Partial<Query>>(
        querySetter,
        queryRef.current ?? {},
      );
      queryRef.current = query;
      try {
        if (beforeRetrieve != null) {
          const result = beforeRetrieve({ query });
          if (result == false) return;
          if (result !== true) {
            onChange(successResult(`retrieved ${name}`, result), query);
            return;
          }
        }
        if (client == null)
          return onChange(nullClientResult<Data>(`retrieve ${name}`), query);
        onChange((p) => loadingResult(`retrieving ${name}`, p.data), query);
        if (signal?.aborted) return;
        const params = { client, query, store };
        listeners.cleanup();
        listeners.set(mountListeners?.({ ...params, onChange: handleListenerChange }));
        const value = await retrieve(params);
        if (signal?.aborted) return;
        onChange(successResult<Data>(`retrieved ${name}`, value), query);
      } catch (error) {
        if (signal?.aborted) return;
        const res = errorResult(`retrieve ${name}`, error);
        if (addStatusOnFailure) addStatus(res.status);
        onChange(res, query);
      }
    },
    [client, name, beforeRetrieve, addStatusOnFailure],
  );
  const retrieveSync = useCallback(
    (query: state.SetArg<Query, Partial<Query>>, options?: core.FetchOptions) =>
      void retrieveAsync(query, options),
    [retrieveAsync],
  );
  return {
    retrieve: retrieveSync,
    retrieveAsync,
  };
};

const useDirect = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  query,
  ...restParams
}: UseDirectRetrieveParams<Query, Data> &
  CreateRetrieveParams<Query, Data, ScopedStore>): UseDirectRetrieveReturn<Data> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restParams);
  const memoquery = useMemoDeepEqual(query);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoquery, { signal }),
    [retrieveAsync, memoquery],
  );
  return rest;
};

const useEffect = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  query,
  onChange,
  ...restParams
}: UseRetrieveEffectParams<Query, Data> &
  CreateRetrieveParams<Query, Data, ScopedStore>): void => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const { retrieveAsync } = useObservableBase<Query, Data, ScopedStore>({
    ...restParams,
    onChange: useCallback(
      (setter, query: Query) => {
        resultRef.current = state.executeSetter(setter, resultRef.current);
        onChange?.(resultRef.current, query);
      },
      [onChange],
    ),
  });
  const memoquery = useMemoDeepEqual(query);
  useAsyncEffect(
    async (signal) => {
      if (memoquery == null) return;
      await retrieveAsync(memoquery, { signal });
    },
    [retrieveAsync, memoquery],
  );
};

export const useObservableRetrieve = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  onChange,
  ...restParams
}: UseRetrieveObservableParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    ScopedStore
  >): UseRetrieveObservableReturn<Query> => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const handleChange = useCallback(
    (setter: state.SetArg<Result<Data>>, query: Query) => {
      resultRef.current = state.executeSetter(setter, resultRef.current);
      onChange?.(resultRef.current, query);
    },
    [onChange],
  );
  return useObservableBase<Query, Data, ScopedStore>({
    ...restParams,
    onChange: handleChange,
  });
};

export const createRetrieve = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store = {},
>(
  createParams: CreateRetrieveParams<Query, Data, ScopedStore>,
): CreateRetrieveReturn<Query, Data> => ({
  useRetrieve: (
    query: Query,
    opts?: Pick<UseDirectRetrieveParams<Query, Data>, "beforeRetrieve">,
  ) => useDirect({ ...createParams, query, ...opts }),
  useRetrieveStateful: () => useStateful(createParams),
  useRetrieveEffect: (Params: UseRetrieveEffectParams<Query, Data>) =>
    useEffect({ ...createParams, ...Params }),
  useRetrieveObservable: (params: UseRetrieveObservableParams<Query, Data>) =>
    useObservableRetrieve({ ...params, ...createParams }),
});
