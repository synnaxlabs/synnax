import { type channel, type Synnax } from "@synnaxlabs/client";
import { id, type MultiSeries, type primitive, status, TimeStamp } from "@synnaxlabs/x";

import { useAsyncEffect } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import { type Sync } from "@/query/sync";
import { state } from "@/state";

export type Params = Record<string, primitive.Value>;

interface ListenerExtraArgs<P extends Params, V extends state.State> {
  params: P;
  client: Synnax;
  onChange: state.Setter<V>;
}

export interface ListenerConfig<P extends Params, V extends state.State> {
  channel: channel.Name;
  onChange: Sync.ListenerHandler<MultiSeries, ListenerExtraArgs<P, V>>;
}

export interface RetrieveArgs<P extends Params> {
  client: Synnax;
  params: P;
}

export type Result<V> =
  | (status.Status<undefined, "loading"> & {
      data: null;
      error: null;
    })
  | (status.Status<status.ExceptionDetails, "error"> & {
      data: null;
      error: unknown;
    })
  | (status.Status<undefined, "success"> & {
      data: V;
      error: null;
    });

export const loadingResult = <V extends state.State>(name: string): Result<V> => ({
  ...status.create<undefined, "loading">({
    key: id.create(),
    variant: "loading",
    message: `Loading ${name}`,
    time: TimeStamp.now(),
  }),
  data: null,
  error: null,
});

export const successResult = <V extends state.State>(
  name: string,
  value: V,
): Result<V> => ({
  ...status.create<undefined, "success">({
    key: id.create(),
    variant: "success",
    message: `Loaded ${name}`,
    time: TimeStamp.now(),
  }),
  data: value,
  error: null,
});

export const errorResult = <V extends state.State>(
  name: string,
  error: unknown,
): Result<V> => ({
  ...status.fromException(error, `Failed to load ${name}`),
  data: null,
  error,
});

interface UseBaseArgs<P extends Params, V extends state.State> {
  retrieve: (args: RetrieveArgs<P>) => Promise<V>;
  listeners: ListenerConfig<P, V>[];
  name: string;
  params: P;
  onChange: state.Setter<Result<V>>;
  client: Synnax | null;
  addListener: Sync.ListenerAdder;
}

export const useBase = <P extends Params, V extends state.State>({
  retrieve,
  listeners,
  name,
  params,
  onChange,
  client,
  addListener,
}: UseBaseArgs<P, V>): void => {
  const memoParams = useMemoDeepEqual(params);
  useAsyncEffect(
    async (signal) => {
      try {
        if (client == null) return;
        onChange(loadingResult(name));
        const value = await retrieve({ client, params });
        if (signal.aborted) return;
        if (listeners.length === 0) {
          onChange(successResult(name, value));
          return;
        }
        const destructors = listeners.map(
          ({ channel, onChange: listenerOnChange }, i) =>
            addListener({
              channel,
              onOpen: () => {
                if (i === listeners.length - 1) onChange(successResult(name, value));
              },
              handler: (frame) => {
                void (async () => {
                  try {
                    await listenerOnChange({
                      client,
                      params,
                      changed: frame.get(channel),
                      onChange: (value) => {
                        onChange((prev) =>
                          successResult(
                            name,
                            state.executeSetter(value, prev.data as unknown as V),
                          ),
                        );
                      },
                    });
                  } catch (error) {
                    onChange(errorResult(name, error));
                  }
                })();
              },
            }),
        );
        return () => destructors.forEach((d) => d());
      } catch (error) {
        onChange(errorResult(name, error));
      }
      return () => {};
    },
    [client, memoParams],
  );
};
