// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DisconnectedError, type Synnax } from "@synnaxlabs/client";
import { type MultiSeries, type primitive, status } from "@synnaxlabs/x";

import { useAsyncEffect } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import { Sync } from "@/query/sync";
import { state } from "@/state";

export type Params = Record<string, primitive.Value>;

interface ListenerExtraArgs<QueryParams extends Params, Data extends state.State> {
  params: QueryParams;
  client: Synnax;
  onChange: state.Setter<Data>;
}

export interface ListenerConfig<QueryParams extends Params, Data extends state.State> {
  channel: channel.Name;
  onChange: Sync.ListenerHandler<MultiSeries, ListenerExtraArgs<QueryParams, Data>>;
}

export interface RetrieveArgs<QueryParams extends Params> {
  client: Synnax;
  params: QueryParams;
}

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

export const errorResult = <Data extends state.State>(
  name: string,
  error: unknown,
): Result<Data> => ({
  ...status.fromException(error, `Failed to load ${name}`),
  data: null,
  error,
});

export interface UseBaseArgs<QueryParams extends Params, Data extends state.State> {
  retrieve: (args: RetrieveArgs<QueryParams>) => Promise<Data>;
  listeners?: ListenerConfig<QueryParams, Data>[];
  name: string;
  params: QueryParams;
  onChange: state.Setter<Result<Data>>;
  client: Synnax | null;
}

export const useBase = <QueryParams extends Params, Data extends state.State>({
  retrieve,
  listeners,
  name,
  params,
  onChange,
  client,
}: UseBaseArgs<QueryParams, Data>): void => {
  const memoParams = useMemoDeepEqual(params);
  const addListener = Sync.useAddListener();
  useAsyncEffect(
    async (signal) => {
      try {
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
        if (signal.aborted) return;
        if (listeners == null || listeners.length === 0)
          return onChange(successResult(name, value));
        const destructors = listeners.map(
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
                        onChange((prev) =>
                          successResult(
                            name,
                            state.executeSetter(value, prev.data as unknown as Data),
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
    },
    [client, memoParams, addListener],
  );
};
