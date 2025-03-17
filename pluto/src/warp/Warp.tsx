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
import { Warp } from "@/warp";

export interface Consumer<T = unknown> {
  channels: channel.Keys;
  decode: (fr: framer.Frame) => Promise<[T | null, boolean]>;
  onChange: (value: T | null) => void;
}

interface ContextValue {
  bindConsumer: (props: Consumer) => Destructor;
}

const Context = createContext<ContextValue>({ bindConsumer: () => () => {} });

export interface ProviderProps extends PropsWithChildren {}

export const Provider = ({ children }: ProviderProps) => {
  const client = PSynnax.use();
  const streamerRef = useRef<framer.Streamer | null>(null);
  const consumers = useRef<Set<Consumer>>(new Set());

  const handleFrame = useCallback((frame: framer.Frame) => {
    consumers.current.forEach(({ decode, onChange }) => {
      void decode(frame).then(([value, shouldChange]) => {
        if (shouldChange) onChange(value);
      });
    });
  }, []);

  const bindConsumer = useCallback((props: Consumer) => {
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

  const ctxValue = useMemo(() => ({ bindConsumer }), [bindConsumer]);

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

export interface Decoder<T> {
  (args: DecodeArgs<T>): Promise<[T | null, boolean]>;
}

export interface Retrieve<T> {
  (props: RetrieveArgs): Promise<T>;
}

export interface RetrieveChannels {
  (props: ChannelsArgs): Promise<channel.Channel[]>;
}

export interface UseRetrieveArgs<T> {
  queryKey: Primitive[];
  initialValue?: T;
  retrieve: Retrieve<T>;
  retrieveChannels: RetrieveChannels;
  decode: Decoder<T>;
}

export interface RetrieveState<T> {
  value: T | null;
  isLoading: boolean;
  error: Error | null;
}

export type UseRetrieveReturn<T> = RetrieveState<T>;

export interface UseRetrieveOnChangeArgs<T>
  extends Omit<UseRetrieveArgs<T>, "initialValue"> {
  onChange: (value: RetrieveState<T>) => void;
}

export const useRetrieveOnChange = <T,>({
  retrieve,
  retrieveChannels,
  decode,
  onChange,
}: UseRetrieveOnChangeArgs<T>): void => {
  const client = PSynnax.use();
  const { bindConsumer } = useContext();
  useAsyncEffect(async () => {
    if (client == null) {
      onChange({ value: null, isLoading: false, error: null });
      return;
    }
    const value = await retrieve({ client });
    onChange({ value, isLoading: false, error: null });
    const channels = await retrieveChannels({ client });
    bindConsumer({
      channels: channels.map((c) => c.key),
      decode: (fr) => decode({ fr, channels, client, current: value }),
      onChange: (v) => onChange({ value: v as T, isLoading: false, error: null }),
    });
  }, [client?.key]);
};

export const useRetrieve = <T,>({
  retrieve,
  retrieveChannels,
  decode,
  queryKey,
  initialValue,
}: UseRetrieveArgs<T>): RetrieveState<T> => {
  const [returnVal, setReturnVal] = useState<RetrieveState<T>>({
    value: initialValue ?? null,
    isLoading: true,
    error: null,
  });
  useRetrieveOnChange({
    retrieve,
    retrieveChannels,
    decode,
    onChange: (v) => setReturnVal(v),
    queryKey,
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
    UseRetrieveArgs<z.output<Z> | null> {
  name: string;
  applyObservable?: (props: ApplyObservableProps<Z, O>) => void;
  applyChanges?: (props: SyncLocalProps<Z>) => Promise<void>;
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
  ...rest
}: UseFormProps<Z, O>): Form.UseReturn<Z> => {
  const client = PSynnax.use();
  const handleError = Status.useErrorHandler();
  const methods = Form.use({
    values: initialValues,
    ...rest,
    sync: false,
    onChange: (props) => {
      if (client == null) return;
      handleError(async () => {
        await applyChanges?.({ ...props, client });
      }, `Failed to apply changes for ${name}`);
    },
  });
  Warp.useRetrieveOnChange<O | null>({
    retrieve,
    retrieveChannels,
    decode,
    onChange: ({ value }) => {
      if (value != null) applyObservable?.({ changes: value, ctx: methods });
    },
    queryKey,
  });
  return methods;
};
