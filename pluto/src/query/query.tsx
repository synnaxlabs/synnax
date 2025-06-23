// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type MultiSeries, type Synnax } from "@synnaxlabs/client";
import { array, type primitive, type status } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { type z } from "zod/v4";

import { Form } from "@/form";
import { useAsyncEffect } from "@/hooks";
import { useMemoPrimitiveArray } from "@/memo";
import { useAddListener } from "@/query/Context";
import { state } from "@/state";
import { Status } from "@/status";
import { Synnax as PSynnax } from "@/synnax";

export type Params = primitive.Value | primitive.Value[];

export type UseReturn<V> =
  | {
      status: "loading";
      message: string;
      data: null;
      error: null;
      statusContent: ReactElement;
    }
  | {
      status: "success";
      message: string;
      data: V;
      error: null;
      statusContent: ReactElement;
    }
  | {
      status: "error";
      data: null;
      message: string;
      error: unknown;
      statusContent: ReactElement;
    };

export interface ListenerArgs<P extends Params, Changed, Value extends state.State> {
  client: Synnax;
  params: P;
  changed: Changed;
  onChange: state.Setter<Value>;
}

export interface ListenerHandler<P extends Params, Changed, Value extends state.State> {
  (args: ListenerArgs<P, Changed, Value>): Promise<void>;
}

export interface ListenerConfig<P extends Params, V extends state.State> {
  channel: channel.Name;
  onChange: ListenerHandler<P, MultiSeries, V>;
}

interface QueryArgs<P extends Params> {
  client: Synnax;
  params: P;
}

export interface CreateArgs<P extends Params, V extends state.State> {
  name: string;
  queryFn: (args: QueryArgs<P>) => Promise<V>;
  listeners: ListenerConfig<P, V>[];
}

export interface QueryHook<P extends Params, V extends state.State> {
  (params: P): UseReturn<V>;
}

export const createStatusContent = (
  status: status.Variant,
  message: string,
): ReactElement => (
  <Status.Text.Centered level="h4" variant={status}>
    {message}
  </Status.Text.Centered>
);

export const create =
  <P extends Params, V extends state.State>({
    queryFn,
    listeners,
    name,
  }: CreateArgs<P, V>): QueryHook<P, V> =>
  (params: P) => {
    const [result, setResult] = useState<UseReturn<V>>({
      status: "loading",
      message: `Loading ${name}`,
      data: null,
      error: null,
      statusContent: createStatusContent("loading", "Loading..."),
    });
    const client = PSynnax.use();
    const addListener = useAddListener();
    const memoParams = useMemoPrimitiveArray(array.toArray(params));
    const handleError = Status.useErrorHandler();

    useAsyncEffect(
      async (signal) => {
        try {
          if (client == null) return;
          setResult({
            status: "loading",
            message: `Loading ${name}`,
            data: null,
            error: null,
            statusContent: createStatusContent("loading", "Loading..."),
          });
          const value = await queryFn({ client, params });
          if (signal.aborted) return;
          setResult({
            status: "success",
            message: `Loaded ${name}`,
            data: value,
            error: null,
            statusContent: createStatusContent("success", "Success!"),
          });
          const destructors = listeners.map(({ channel, onChange }) =>
            addListener({
              channels: channel,
              handler: (frame) => {
                handleError(
                  async () =>
                    await onChange({
                      client,
                      params,
                      changed: frame.get(channel),
                      onChange: (value) =>
                        setResult((prev) => ({
                          ...prev,
                          error: null,
                          status: "success",
                          data: state.executeSetter(value, prev.data as unknown as V),
                          statusContent: createStatusContent("success", "Success!"),
                        })),
                    }),
                );
              },
            }),
          );
          return () => destructors.forEach((d) => d());
        } catch (error) {
          setResult({
            status: "error",
            message: `Failed to load ${name}`,
            data: null,
            error,
            statusContent: createStatusContent("error", "Error!"),
          });
        }
        return () => {};
      },
      [memoParams, client],
    );
    return result;
  };

interface MutateArgs<K extends primitive.Value, Z extends z.ZodObject> {
  key: K;
  client: Synnax;
  values: z.infer<Z>;
}

export interface CreateFormArgs<K extends primitive.Value, Z extends z.ZodObject> {
  name: string;
  schema: Z;
  queryFn: (args: QueryArgs<K | undefined>) => Promise<z.infer<Z> | null>;
  mutationFn: (args: MutateArgs<K, Z>) => Promise<unknown>;
  listeners: ListenerConfig<K, z.infer<Z>>[];
}

export type UseFormReturn<Z extends z.ZodType> = {
  status: "loading" | "success" | "error";
  message: string;
  error: null;
  statusContent: ReactElement;
  form: Form.UseReturn<Z>;
  save: () => void;
};

interface UseFormArgs<K extends primitive.Value, Z extends z.ZodObject> {
  key?: K;
  initialValues: z.infer<Z>;
  autoSave?: boolean;
}

export interface FormHook<K extends primitive.Value, Z extends z.ZodObject> {
  ({ key, initialValues, autoSave }: UseFormArgs<K, Z>): UseFormReturn<Z>;
}

export const createForm =
  <K extends primitive.Value, Z extends z.ZodObject>({
    name,
    schema,
    queryFn,
    listeners,
  }: CreateFormArgs<K, Z>): FormHook<K, Z> =>
  ({ key, initialValues }: UseFormArgs<K, Z>): UseFormReturn<Z> => {
    const [status, setStatus] = useState<Omit<UseFormReturn<Z>, "form" | "save">>({
      status: "loading",
      message: `Loading ${name}`,
      error: null,
      statusContent: createStatusContent("loading", "Loading..."),
    });
    const form = Form.use<Z>({ schema, values: initialValues });
    const client = PSynnax.use();
    const addListener = useAddListener();
    const handleError = Status.useErrorHandler();

    useAsyncEffect(
      async (signal) => {
        try {
          if (client == null) return;
          setStatus({
            status: "loading",
            message: `Loading ${name}`,
            error: null,
            statusContent: createStatusContent("loading", "Loading..."),
          });
          const value = await queryFn({ client, params: key });
          if (signal.aborted) return;
          form.set("", value);
          form.setCurrentStateAsInitialValues();
          setStatus({
            status: "success",
            message: `Loaded ${name}`,
            error: null,
            statusContent: createStatusContent("success", "Success!"),
          });
          const destructors = listeners.map(({ channel, onChange }) =>
            addListener({
              channels: channel,
              handler: (frame) => {
                handleError(
                  async () =>
                    await onChange({
                      client,
                      params: key ?? (undefined as K),
                      changed: frame.get(channel),
                      onChange: (value) => {
                        form.set("", value);
                        form.setCurrentStateAsInitialValues();
                        setStatus((prev) => ({
                          ...prev,
                          status: "success",
                          statusContent: createStatusContent("success", "Success!"),
                        }));
                      },
                    }),
                );
              },
            }),
          );
          return () => destructors.forEach((d) => d());
        } catch (error) {
          setStatus({
            status: "error",
            message: `Failed to load ${name}`,
            error: error as null,
            statusContent: createStatusContent("error", "Error!"),
          });
        }
        return () => {};
      },
      [client],
    );
    return {
      status: status.status,
      error: status.error,
      message: status.message,
      statusContent: status.statusContent,
      form,
      save: () => {},
    };
  };
