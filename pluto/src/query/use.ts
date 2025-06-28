import { type channel, DisconnectedError, type Synnax } from "@synnaxlabs/client";
import {
  type Destructor,
  type MultiSeries,
  type primitive,
  status,
} from "@synnaxlabs/x";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useAsyncEffect } from "@/hooks";
import { Sync } from "@/query/sync";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

/**
 * Parameters used to retrieve and or/update a resource from within a query. The query
 * re-executes whenever the parameters change.
 */
export type Params = Record<string, primitive.Value>;

interface ListenerExtraArgs<QueryParams extends Params, Data extends state.State> {
  params: QueryParams;
  client: Synnax;
  onChange: state.Setter<Data>;
}

/**
 * Configuration for a listener that is called whenever a new value is received
 * from the specified channel. The listener is called with the new value and can
 * choose to update the state of the query by calling the `onChange` function.
 *
 * The listener will not be called if the query is in a loading or an error state.
 */
export interface ListenerConfig<QueryParams extends Params, Data extends state.State> {
  /** The channel to listen to. */
  channel: channel.Name;
  /** The function to call when a new value is received. */
  onChange: Sync.ListenerHandler<MultiSeries, ListenerExtraArgs<QueryParams, Data>>;
}

/**
 * Arguments passed to the `retrieve` function on the query.
 */
export interface RetrieveArgs<QueryParams extends Params> {
  client: Synnax;
  params: QueryParams;
}

/**
 * The result of a query. The query can be in one of three states:
 * - `loading` - Data is currently being fetched.
 * - `error` - An error occurred while fetching data, or while handling values provided
 * to a listener.
 * - `success` - Data was successfully fetched and is available in the `data` field.
 */
export type Result<Data extends state.State> =
  | (status.Status<undefined, "loading"> & {
      data: null;
      error: null;
    })
  | (status.Status<status.ExceptionDetails, "error"> & {
      data: null;
      error: unknown;
    })
  | (status.Status<undefined, "success"> & {
      data: Data;
      error: null;
    });

/** A factory function to create a loading result. */
export const loadingResult = <Data extends state.State>(
  name: string,
): Result<Data> => ({
  ...status.create<undefined, "loading">({
    variant: "loading",
    message: `Loading ${name}`,
  }),
  data: null,
  error: null,
});

/** A factory function to create a success result. */
export const successResult = <Data extends state.State>(
  name: string,
  value: Data,
): Result<Data> => ({
  ...status.create<undefined, "success">({
    variant: "success",
    message: `Loaded ${name}`,
  }),
  data: value,
  error: null,
});

/** A factory function to create an error result. */
export const errorResult = <Data extends state.State>(
  name: string,
  error: unknown,
): Result<Data> => ({
  ...status.fromException(error, `Failed to load ${name}`),
  data: null,
  error,
});

/**
 * Arguments passed to the `useBase` hook.
 * @template QParams - The type of the parameters for the query.
 * @template Data - The type of the data being retrieved.
 */
export interface CreateArgs<QParams extends Params, Data extends state.State> {
  /**
   * The name of the resource being retrieve. This is used to make pretty messages for
   * the various query states.
   */
  name: string;
  /** Executed when the query is first created, or whenever the query parameters change. */
  retrieve: (args: RetrieveArgs<QParams>) => Promise<Data>;
  /**
   * Listeners to mount to the query. These listeners will be re-mounted when
   * the query parameters changed and/or the client disconnects/re-connects or clusters
   * are switched.
   *
   * These listeners will NOT be remounted when the identity of the onChange function
   * changes.
   */
  listeners?: ListenerConfig<QParams, Data>[];
}

export interface UseObservableArgs<P extends Params, V extends state.State> {
  onChange: state.Setter<Result<V>>;
}

export interface UseObservableReturn<P extends Params> {
  retrieve: (params: P, options: { signal?: AbortSignal }) => void;
  retrieveAsync: (params: P, options: { signal?: AbortSignal }) => Promise<void>;
}

export interface UseStatefulArgs<P extends Params, V extends state.State>
  extends CreateArgs<P, V> {}

export type UseStatefulReturn<P extends Params, V extends state.State> = Result<V> & {
  retrieve: (params: P, options: { signal?: AbortSignal }) => void;
  retrieveAsync: (params: P, options: { signal?: AbortSignal }) => Promise<void>;
};

export interface UseDirectArgs<P extends Params> {
  params: P;
}

export type UseDirectReturn<V extends state.State> = Result<V>;

export interface UseEffectArgs<P extends Params, V extends state.State> {
  params: P;
  onChange: state.Setter<V>;
}

export interface CreateReturn<P extends Params, V extends state.State> {
  observable: (args: UseObservableArgs<P, V>) => UseObservableReturn<P>;
  stateful: (args: UseStatefulArgs<P, V>) => UseStatefulReturn<P, V>;
  direct: (args: UseDirectArgs<P>) => UseDirectReturn<V>;
  effect: (args: UseEffectArgs<P, V>) => void;
}

export interface Create<P extends Params, V extends state.State> {
  (args: CreateArgs<P, V>): CreateReturn<P, V>;
}

export const useObservable = <QueryParams extends Params, Data extends state.State>({
  retrieve,
  listeners,
  name,
  onChange,
}: UseObservableArgs<QueryParams, Data> &
  CreateArgs<QueryParams, Data>): UseObservableReturn<QueryParams> => {
  const addListener = Sync.useAddListener();
  const destructorsRef = useRef<Destructor[]>([]);
  const cleanupDestructors = useCallback(() => {
    destructorsRef.current.forEach((d) => d());
  }, []);
  useEffect(() => cleanupDestructors, [cleanupDestructors]);
  const client = PSynnax.use();
  const base = useCallback(
    async (params: QueryParams, { signal }: { signal?: AbortSignal }) => {
      try {
        cleanupDestructors();
        if (client == null)
          return onChange(
            errorResult(
              name,
              new DisconnectedError(
                `Cannot retrieve ${name} because no cluster is connected.`,
              ),
            ),
          );
        onChange(loadingResult(name));
        const value = await retrieve({ client, params });
        if (signal?.aborted) return;
        if (listeners == null || listeners.length === 0)
          return onChange(successResult(name, value));
        destructorsRef.current = listeners.map(
          ({ channel, onChange: listenerOnChange }, i) =>
            addListener({
              channel,
              onOpen: () =>
                i === listeners.length - 1 && onChange(successResult(name, value)),
              handler: (frame) => {
                void (async () => {
                  try {
                    await listenerOnChange({
                      client,
                      params,
                      changed: frame.get(channel),
                      onChange: (value) => {
                        onChange((prev) => {
                          if (prev.data == null) return prev;
                          const next = state.executeSetter(value, prev.data);
                          return successResult(name, next);
                        });
                      },
                    });
                  } catch (error) {
                    onChange(errorResult(name, error));
                  }
                })();
              },
            }),
        );
      } catch (error) {
        onChange(errorResult(name, error));
      }
    },
    [client, name, addListener],
  );
  return {
    retrieve: (params: QueryParams, options: { signal?: AbortSignal }) => {
      void base(params, options);
    },
    retrieveAsync: async (params: QueryParams, options: { signal?: AbortSignal }) =>
      await base(params, options),
  };
};

const useStateful = <P extends Params, V extends state.State>(
  args: UseStatefulArgs<P, V> & CreateArgs<P, V>,
): UseStatefulReturn<P, V> => {
  const [state, setState] = useState<Result<V>>(loadingResult(args.name));
  return {
    ...state,
    ...useObservable({ ...args, onChange: setState }),
  };
};

export const useDirect = <P extends Params, V extends state.State>({
  params,
  ...restArgs
}: UseDirectArgs<P> & CreateArgs<P, V>): UseDirectReturn<V> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restArgs);
  const memoParams = useMemo(() => params, [params]);
  useAsyncEffect(
    async (signal) => {
      await retrieveAsync(memoParams, { signal });
    },
    [retrieveAsync, memoParams],
  );
  return rest;
};

const useEffect = <P extends Params, V extends state.State>({
  params,
  ...restArgs
}: UseEffectArgs<P, V> & CreateArgs<P, V>): void => {
  const { retrieveAsync } = useObservable<P, V>(restArgs);
  const memoParams = useMemo(() => params, [params]);
  useAsyncEffect(
    async (signal) => {
      await retrieveAsync(memoParams, { signal });
    },
    [retrieveAsync, memoParams],
  );
};

export const create = <P extends Params, V extends state.State>(
  factoryArgs: CreateArgs<P, V>,
): CreateReturn<P, V> => ({
  observable: (args: UseObservableArgs<P, V>) =>
    useObservable({ ...factoryArgs, ...args }),
  stateful: (args: UseStatefulArgs<P, V>) => useStateful({ ...factoryArgs, ...args }),
  direct: (args: UseDirectArgs<P>) => useDirect({ ...factoryArgs, ...args }),
  effect: (args: UseEffectArgs<P, V>) => useEffect({ ...factoryArgs, ...args }),
});
