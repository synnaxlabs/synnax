import { DisconnectedError, type Synnax } from "@synnaxlabs/client";
import { type Destructor, type MultiSeries, observe, type record } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";

import { type Params } from "@/flux/params";
import { errorResult, loadingResult, type Result, successResult } from "@/flux/result";
import {
  type CreateRetrieveArgs,
  type UseStatefulRetrieveReturn,
} from "@/flux/retrieve";
import { Sync } from "@/flux/sync";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

export type UseListReturn<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> = UseStatefulRetrieveReturn<RetrieveParams, K[]> & {
  useListItem: (key: K) => E | undefined;
};

export interface CreateListArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> extends Omit<CreateRetrieveArgs<RetrieveParams, E[]>, "listeners"> {
  retrieveByKey: (key: K) => Promise<E>;
  listeners?: ListListenerConfig<RetrieveParams, K, E>[];
}

export interface UseList<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  (): UseListReturn<RetrieveParams, K, E>;
}

interface ListListenerExtraArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  params: RetrieveParams;
  client: Synnax;
  onChange: (key: K, e: state.Setter<E | null>) => void;
}

export interface ListListenerConfig<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  channel: string;
  onChange: Sync.ListenerHandler<
    MultiSeries,
    ListListenerExtraArgs<RetrieveParams, K, E>
  >;
}

export const createList =
  <P extends Params, K extends record.Key, E extends record.Keyed<K>>({
    name,
    listeners,
    retrieve,
    retrieveByKey,
  }: CreateListArgs<P, K, E>): UseList<P, K, E> =>
  () => {
    const dataRef = useRef<Map<K, E>>(new Map());
    const listenersRef = useRef<Map<() => void, K>>(new Map());
    const [result, setResult] = useState<Result<K[]>>();
    const addListener = Sync.useAddListener();
    const destructorsRef = useRef<Destructor[]>([]);
    const cleanupDestructors = useCallback(() => {
      destructorsRef.current.forEach((d) => d());
      destructorsRef.current = [];
    }, []);
    useEffect(() => cleanupDestructors, [cleanupDestructors]);
    const client = PSynnax.use();
    const paramsRef = useRef<P | {}>({});
    const base = useCallback(
      async (
        paramsSetter: state.SetArg<P, P | {}>,
        { signal }: { signal?: AbortSignal },
      ) => {
        const params = state.executeSetter(paramsSetter, paramsRef.current);
        paramsRef.current = params;
        try {
          cleanupDestructors();
          if (client == null)
            return setResult(
              errorResult(
                name,
                new DisconnectedError(
                  `Cannot retrieve ${name} because no cluster is connected.`,
                ),
              ),
            );
          setResult(loadingResult(name));
          const value = await retrieve({ client, params });
          const keys = value.map((v) => v.key);
          if (signal?.aborted) return;
          if (listeners == null || listeners.length === 0)
            return setResult(successResult(name, keys));
          destructorsRef.current = listeners.map(
            ({ channel, onChange: listenerOnChange }, i) =>
              addListener({
                channel,
                onOpen: () =>
                  i === listeners.length - 1 && setResult(successResult(name, keys)),
                handler: (frame) => {
                  void (async () => {
                    try {
                      await listenerOnChange({
                        client,
                        params,
                        changed: frame.get(channel),
                        onChange: (k, setter) => {
                          const v = value.find((v) => v.key === k);
                          if (v == null) return;
                          const res = setter(v);
                          if (res == null) return;
                          dataRef.current.set(k, res);
                          listenersRef.current.forEach((key, listener) => {
                            if (key === k) listener(res);
                          });
                        },
                      });
                    } catch (error) {
                      setResult(errorResult(name, error));
                    }
                  })();
                },
              }),
          );
        } catch (error) {
          setResult(errorResult(name, error));
        }
      },
      [client, name, addListener],
    );

    const useListItem = (key: K) =>
      useSyncExternalStore<E | undefined>(
        useCallback(
          (callback) => {
            listenersRef.current.set(callback, key);
            return () => listenersRef.current.delete(callback);
          },
          [key],
        ),
        useCallback(() => {
          const res = dataRef.current.get(key);
          if (res == null) void retrieveByKey(key);
          return res;
        }, [key]),
      );
    const res: Result<K[]> = { ...result, data: result?.data ?? [] } as Result<K[]>;
    const v: UseListReturn<P, K, E> = {
      retrieve: (
        params: state.SetArg<P, P | {}>,
        options: { signal?: AbortSignal },
      ) => {
        void base(params, options);
      },
      retrieveAsync: async (
        params: state.SetArg<P, P | {}>,
        options: { signal?: AbortSignal },
      ) => await base(params, options),
      useListItem,
      ...res,
    };
    return v;
  };
