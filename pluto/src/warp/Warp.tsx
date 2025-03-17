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

type RetrieveState<T> =
  | {
      value: T;
      isLoading: false;
      error: null;
    }
  | {
      value: T | null;
      isLoading: true;
      error: null;
    }
  | {
      value: null;
      isLoading: false;
      error: Error;
    };

export type UseRetrieveReturn<T> = RetrieveState<T>;

export interface UseRetrieveOnChangeArgs<T>
  extends Omit<UseRetrieveArgs<T>, "initialValue"> {
  onChange: (value: RetrieveState<T>) => void;
}

export const useRetrieveListener = <T,>({
  retrieve,
  retrieveChannels,
  decode,
  onChange,
}: UseRetrieveOnChangeArgs<T>): void => {
  const client = PSynnax.use();
  const { bindListener: bindConsumer } = useContext();
  useAsyncEffect(async () => {
    if (client == null)
      return onChange({
        value: null,
        isLoading: false,
        error: new Error("Client not found"),
      });
    try {
      const value = await retrieve({ client });
      onChange({ value, isLoading: false, error: null });
      const channels = await retrieveChannels({ client });
      bindConsumer({
        channels: channels.map((c) => c.key),
        onChange: (fr) => {
          decode({ fr, channels, client, current: value })
            .then(([v, shouldChange]) => {
              if (shouldChange)
                onChange({ value: v as T, isLoading: false, error: null });
            })
            .catch((error) => onChange({ value: null, isLoading: false, error }));
        },
      });
    } catch (error) {
      onChange({ value: null, isLoading: false, error: error as Error });
    }
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
  useRetrieveListener({
    retrieve,
    retrieveChannels,
    decode,
    onChange: setReturnVal,
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
  useRetrieveListener<O | null>({
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
