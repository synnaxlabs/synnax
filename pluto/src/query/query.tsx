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
import { type ReactElement, useCallback, useRef, useState } from "react";
import { type z } from "zod/v4";

import { Form } from "@/form";
import { useAsyncEffect, useCombinedStateAndRef } from "@/hooks";
import { useMemoPrimitiveArray } from "@/memo";
import { Sync } from "@/query/sync";
import { state } from "@/state";
import { Status } from "@/status";
import { Synnax as PSynnax } from "@/synnax";

export type Params = primitive.Value | primitive.Value[];

interface ListenerExtraArgs<P extends Params, V extends state.State> {
  params: P;
  client: Synnax;
  onChange: state.Setter<V>;
}

export interface ListenerConfig<P extends Params, V extends state.State> {
  channel: channel.Name;
  onChange: Sync.ListenerHandler<MultiSeries, ListenerExtraArgs<P, V>>;
}

interface RetrieveArgs<P extends Params> {
  client: Synnax;
  params: P;
}

interface UpdateArgs<P extends Params, V extends state.State> {
  client: Synnax;
  params: P;
  values: V;
}

export const createStatusContent = (
  status: status.Variant,
  message: string,
): ReactElement => (
  <Status.Text.Centered level="h4" variant={status}>
    {message}
  </Status.Text.Centered>
);

export interface UseArgs<P extends Params, V extends state.State> {
  name: string;
  params: P;
  retrieve: (args: RetrieveArgs<P>) => Promise<V>;
  listeners: ListenerConfig<P, V>[];
}

export type UseReturn<V> = {
  message: string;
} & (
  | {
      status: "loading";
      data: null;
      error: null;
    }
  | {
      status: "success";
      data: V;
      error: null;
    }
  | {
      status: "error";
      data: null;
      error: unknown;
    }
);

const initialResult = <V extends state.State>(name: string): UseReturn<V> => ({
  status: "loading",
  message: `Loading ${name}`,
  data: null,
  error: null,
});

const loadingResult = <V extends state.State>(name: string): UseReturn<V> => ({
  status: "loading",
  message: `Loading ${name}`,
  data: null,
  error: null,
});

const successResult = <V extends state.State>(
  name: string,
  value: V,
): UseReturn<V> => ({
  status: "success",
  message: `Loaded ${name}`,
  data: value,
  error: null,
});

const errorResult = <V extends state.State>(
  name: string,
  error: unknown,
): UseReturn<V> => ({
  status: "error",
  message: `Failed to load ${name}`,
  data: null,
  error,
});

export const use = <P extends Params, V extends state.State>({
  retrieve,
  listeners,
  name,
  params,
}: UseArgs<P, V>): UseReturn<V> => {
  const [result, setResult] = useState<UseReturn<V>>(initialResult(name));
  const client = PSynnax.use();
  const addListener = Sync.useAddListener();
  useBase<P, V>({
    retrieve,
    listeners,
    name,
    params,
    client,
    addListener,
    onChange: setResult,
  });
  return result;
};

interface UseBaseArgs<P extends Params, V extends state.State> {
  retrieve: (args: RetrieveArgs<P>) => Promise<V>;
  listeners: ListenerConfig<P, V>[];
  name: string;
  params: P;
  onChange: state.Setter<UseReturn<V>>;
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
  const memoParams = useMemoPrimitiveArray(array.toArray(params));
  useAsyncEffect(
    async (signal) => {
      try {
        if (client == null) return;
        onChange(loadingResult(name));
        const value = await retrieve({ client, params });
        if (signal.aborted) return;
        const destructors = listeners.map(
          ({ channel, onChange: listenerOnChange }, i) =>
            addListener({
              channels: channel,
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
                        onChange((prev) => ({
                          ...prev,
                          error: null,
                          status: "success",
                          data: state.executeSetter(value, prev.data as unknown as V),
                        }));
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

export interface UseFormArgs<K extends primitive.Value, Z extends z.ZodObject>
  extends UseArgs<K, z.infer<Z> | null> {
  initialValues: z.infer<Z>;
  autoSave?: boolean;
  schema: Z;
  update: (args: UpdateArgs<K, z.infer<Z>>) => Promise<z.infer<Z>>;
}

export type UseFormReturn<Z extends z.ZodType> = Omit<UseReturn<z.infer<Z>>, "data"> & {
  form: Form.UseReturn<Z>;
  save: () => void;
};

export const useForm = <K extends primitive.Value, Z extends z.ZodObject>({
  name,
  params,
  initialValues,
  schema,
  retrieve,
  listeners,
  update,
}: UseFormArgs<K, Z>): UseFormReturn<Z> => {
  const [status, setStatus, statusRef] = useCombinedStateAndRef<
    UseReturn<z.infer<Z> | null>
  >(loadingResult(name));
  const form = Form.use<Z>({ schema, values: initialValues });
  const client = PSynnax.use();
  const addListener = Sync.useAddListener();

  const handleResultChange: state.Setter<UseReturn<z.infer<Z> | null>> = (setter) => {
    const nextStatus = state.executeSetter(setter, statusRef.current);
    form.set("", nextStatus.data);
    form.setCurrentStateAsInitialValues();
    setStatus(nextStatus);
  };

  const abortControllerRef = useRef<AbortController | null>(null);

  const memoParams = useMemoPrimitiveArray(array.toArray(params));

  const handleSave = useCallback(() => {
    if (abortControllerRef.current != null) abortControllerRef.current.abort();
    abortControllerRef.current = new AbortController();
    void (async () => {
      try {
        if (client == null) return;
        const res = await update({ client, params, values: form.value() });
        if (abortControllerRef.current?.signal.aborted) return;
        form.set("", res);
        form.setCurrentStateAsInitialValues();
        handleResultChange(successResult(name, res));
      } catch (error) {
        setStatus(errorResult(name, error));
      }
    })();
  }, [client, form, name, memoParams, update]);

  useBase({
    retrieve,
    listeners,
    name,
    params,
    onChange: handleResultChange,
    client,
    addListener,
  });

  return {
    form,
    save: handleSave,
    ...status,
  };
};
