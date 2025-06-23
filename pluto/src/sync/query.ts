// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type MultiSeries, type Synnax } from "@synnaxlabs/client";
import { array, type primitive } from "@synnaxlabs/x";
import { useState } from "react";

import { useAsyncEffect } from "@/hooks";
import { useMemoPrimitiveArray } from "@/memo";
import { Status } from "@/status";
import { useAddListener } from "@/sync/Context";
import { Synnax as PSynnax } from "@/synnax";

export type Params = primitive.Value | primitive.Value[];

export type UseQueryReturn<V> =
  | {
      status: "loading";
      data: null;
    }
  | {
      status: "success";
      data: V;
    }
  | {
      status: "error";
      message: string;
      error: unknown;
    };

export interface ListenerContext<P extends Params, Changed, Value> {
  client: Synnax;
  params: P;
  changed: Changed;
  onChange: (value: Value) => void;
}

export interface ListenerHandler<P extends Params, Changed, Value> {
  (context: ListenerContext<P, Changed, Value>): Promise<void>;
}

type ListenerConfig<P extends Params, V> = {
  channel: channel.Name;
  onChange: ListenerHandler<P, MultiSeries, V>;
};

export interface CreateQueryArgs<P extends Params, V> {
  queryFn: (client: Synnax, args: P) => Promise<V>;
  listeners: ListenerConfig<P, V>[];
}

export interface QueryHook<P extends Params, V> {
  (params: P): UseQueryReturn<V>;
}

export const createQuery =
  <P extends Params, V>({
    queryFn,
    listeners,
  }: CreateQueryArgs<P, V>): QueryHook<P, V> =>
  (params: P) => {
    const [result, setResult] = useState<UseQueryReturn<V>>({
      status: "loading",
      data: null,
    });
    const client = PSynnax.use();
    const addListener = useAddListener();
    const memoParams = useMemoPrimitiveArray(array.toArray(params));
    const handleError = Status.useErrorHandler();

    useAsyncEffect(
      async (signal) => {
        try {
          if (client == null) return;
          setResult({ status: "loading", data: null });
          const value = await queryFn(client, params);
          if (signal.aborted) return;
          setResult({ status: "success", data: value });
          const destructors = listeners.map(({ channel, onChange }) =>
            addListener({
              channels: channel,
              handler: (frame) => {
                handleError(async () => {
                  const value = await onChange({ client, params }, frame.get(channel));
                  if (value == null) return;
                  setResult({ status: "success", data: value });
                });
              },
            }),
          );
          return () => destructors.forEach((d) => d());
        } catch (error) {
          setResult({ status: "error", message: "Failed to query data", error });
        }
        return () => {};
      },
      [memoParams, client],
    );
    return result;
  };
