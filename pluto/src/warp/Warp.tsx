// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer, type Synnax } from "@synnaxlabs/client";
import { type Destructor, type Primitive } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext as reactUseContext,
  useMemo,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import { Form } from "@/form";
import { useAsyncEffect } from "@/hooks";
import { Status } from "@/status";
import { Synnax as PSynnax } from "@/synnax";

export interface Listener {
  channels: channel.Keys;
  onChange: (fr: framer.Frame) => void;
}

interface ContextValue {
  bindListener: (props: Listener) => Destructor;
}

const Context = createContext<ContextValue>({ bindListener: () => () => {} });

export interface ProviderProps extends PropsWithChildren {}

export const Provider = ({ children }: ProviderProps) => {
  const client = PSynnax.use();
  const streamerRef = useRef<framer.Streamer | null>(null);
  const consumers = useRef<Set<Listener>>(new Set());

  const handleFrame = useCallback((frame: framer.Frame) => {
    consumers.current.forEach(({ onChange }) => onChange(frame));
  }, []);

  const bindListener = useCallback((props: Listener) => {
    consumers.current.add(props);
    return () => consumers.current.delete(props);
  }, []);

  useAsyncEffect(async () => {
    if (client == null) return;
    const streamer = await client.openStreamer([]);
    streamerRef.current = streamer;
    const observable = new framer.ObservableStreamer(streamer);
    const disconnect = observable.onChange(handleFrame);
    return async () => {
      disconnect();
      await observable.close();
    };
  }, [client?.key]);

  const ctxValue = useMemo(() => ({ bindListener }), [bindListener]);
  return <Context.Provider value={ctxValue}>{children}</Context.Provider>;
};

const useContext = () => reactUseContext(Context);

interface RetrieveArgs {
  client: Synnax;
}

interface ChannelsArgs {
  client: Synnax;
}

interface DecodeArgs<T> {
  fr: framer.Frame;
  channels: channel.Channel[];
  current: T | null;
  client: Synnax;
}

export interface Decoder<I, O = I> {
  (args: DecodeArgs<I>): Promise<[O | null, boolean]>;
}

export interface Retrieve<T> {
  (props: RetrieveArgs): Promise<T>;
}

export interface RetrieveChannels {
  (props: ChannelsArgs): Promise<channel.Channel[]>;
}

export interface UseRetrieveArgs<Value, InitialValue = undefined> {
  queryKey: Primitive[];
  initialValue: InitialValue;
  retrieve: Retrieve<Value>;
  retrieveChannels: RetrieveChannels;
  decode: Decoder<Value>;
}

export type RetrieveState<Value, InitialValue = null> =
  | {
      value: Value;
      isLoading: false;
      error: null;
    }
  | {
      value: InitialValue;
      isLoading: true;
      error: null;
    }
  | {
      value: InitialValue;
      isLoading: false;
      error: Error;
    };

export type UseRetrieveReturn<Value, InitialValue = undefined> = RetrieveState<
  Value,
  InitialValue
>;

export interface UseRetrieveOnChangeArgs<Value, InitialValue = undefined>
  extends UseRetrieveArgs<Value, InitialValue> {
  onChange: (value: RetrieveState<Value, InitialValue>) => void;
}

export const useRetrieveListener = <Value, InitialValue = undefined>({
  retrieve,
  retrieveChannels,
  decode,
  onChange,
  initialValue,
}: UseRetrieveOnChangeArgs<Value, InitialValue>): void => {
  const client = PSynnax.use();
  const { bindListener: bindConsumer } = useContext();
  useAsyncEffect(async () => {
    if (client == null)
      return onChange({
        value: initialValue,
        isLoading: false,
        error: new Error("Client not found"),
      });
    try {
      const value = await retrieve({ client });
      onChange({ value, isLoading: false, error: null });
      const channels = await retrieveChannels({ client });
      return bindConsumer({
        channels: channels.map((c) => c.key),
        onChange: (fr) => {
          decode({ fr, channels, client, current: value })
            .then(([v, shouldChange]) => {
              if (shouldChange)
                onChange({ value: v as Value, isLoading: false, error: null });
            })
            .catch((error) =>
              onChange({ value: initialValue, isLoading: false, error }),
            );
        },
      });
    } catch (error) {
      onChange({ value: initialValue, isLoading: false, error: error as Error });
    }
  }, [client?.key]);
};

export const useRetrieve = <Value, InitialValue = undefined>({
  retrieve,
  retrieveChannels,
  decode,
  queryKey,
  initialValue,
}: UseRetrieveArgs<Value, InitialValue>): RetrieveState<Value, InitialValue> => {
  const [returnVal, setReturnVal] = useState<RetrieveState<Value, InitialValue>>({
    value: initialValue,
    isLoading: true,
    error: null,
  });
  useRetrieveListener({
    retrieve,
    retrieveChannels,
    decode,
    onChange: setReturnVal,
    queryKey,
    initialValue,
  });
  return returnVal;
};

interface ApplyObservableProps<Z extends z.ZodTypeAny, O = Z> {
  changes: O;
  ctx: Form.ContextValue<Z>;
}

interface SyncLocalProps<Z extends z.ZodTypeAny> extends Form.OnChangeProps<Z> {
  client: Synnax;
}

export interface UseFormProps<Z extends z.ZodTypeAny, O = Z>
  extends Form.UseProps<Z>,
    Omit<UseRetrieveArgs<z.output<Z> | null>, "initialValue"> {
  name: string;
  applyObservable?: (props: ApplyObservableProps<Z, O>) => void;
  applyChanges?: (props: SyncLocalProps<Z>) => Promise<void>;
  autoSave?: boolean;
}

export interface UseFormReturn<Z extends z.ZodTypeAny> extends Form.UseReturn<Z> {
  save(): void;
}

export const useForm = <Z extends z.ZodTypeAny, O = Z>({
  name,
  values: initialValues,
  applyChanges,
  applyObservable,
  retrieveChannels,
  retrieve,
  decode,
  queryKey,
  autoSave = true,
  ...rest
}: UseFormProps<Z, O>): UseFormReturn<Z> => {
  const client = PSynnax.use();
  const handleError = Status.useErrorHandler();
  const handleApplyChanges = useCallback(
    (props: Form.OnChangeProps<Z>) => {
      void handleError(async () => {
        if (client == null) return;
        await applyChanges?.({ ...props, client });
      }, `Failed to apply changes for ${name}`);
    },
    [applyChanges, client],
  );

  const methods = Form.use({
    values: initialValues,
    ...rest,
    sync: false,
    onChange: autoSave ? handleApplyChanges : undefined,
  });

  const save = useCallback(() => {
    handleApplyChanges({
      values: methods.value(),
      path: "",
      prev: null,
      valid: true,
    });
  }, [handleApplyChanges, methods]);

  useRetrieveListener<O | null, O | null>({
    retrieve,
    retrieveChannels,
    decode,
    onChange: ({ value }) => {
      if (value != null) applyObservable?.({ changes: value, ctx: methods });
    },
    queryKey,
    initialValue: null,
  });
  return { ...methods, save };
};
