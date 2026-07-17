// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, strings, unique } from "@synnaxlabs/x";
import type z from "zod";

import { type ChannelListener, type StoreConfig, type Stores } from "@/cache/store";
import { type AsyncErrorHandler } from "@/cache/types";
import { NotFoundError } from "@/errors";
import { type framer } from "@/framer";

/**
 * Sorts channel names to ensure deletions are processed before other changes.
 * This ensures that modifications to things like relationships (delete followed by create)
 * are processed in the correct order.
 *
 * @param a - First channel name
 * @param b - Second channel name
 * @returns Sort order (-1, 0, or 1)
 */
const channelNameSort = (a: string, b: string) => {
  const aHasDelete = a.includes("delete");
  const bHasDelete = b.includes("delete");
  if (aHasDelete && !bHasDelete) return -1;
  if (!aHasDelete && bHasDelete) return 1;
  return 0;
};

/** An opened change stream: frame notifications plus close. */
export interface ObservableStream {
  onChange: (handler: (frame: framer.Frame) => void) => void;
  close: () => Promise<void>;
}

/**
 * Hooks passed to a {@link StreamOpener}. The implementation must fire onOpen
 * after the stream is live but before any frame is delivered, and onReopen
 * after every successful reconnect (not the initial open).
 */
export interface StreamOpenerHooks {
  onOpen?: () => void;
  onReopen?: () => void;
}

/**
 * Opens a change stream over the given channels. Injected at construction so
 * the cache carries no dependency on the streaming transport.
 */
export interface StreamOpener {
  (channels: string[], hooks: StreamOpenerHooks): Promise<ObservableStream>;
}

/**
 * Arguments for opening a cache streamer.
 *
 * @template ScopedStores - The type of the stores
 */
export interface StreamerParams<ScopedStores extends Stores> {
  /** Function to handle errors that occur during streaming */
  handleError: AsyncErrorHandler;
  /** Configuration defining store structure and listeners */
  storeConfig: StoreConfig<ScopedStores>;
  /** Function to open the change stream */
  openStreamer: StreamOpener;
  /** The stores to update with streamed data */
  store: ScopedStores;
  /**
   * Called once when the stream first opens, before any frame is processed.
   */
  onOpen?: () => void;
  /**
   * Called after every successful reconnect of the underlying stream (not the
   * initial open). Changes may have been missed while disconnected.
   */
  onReopen?: () => void;
}

/**
 * A lazily-opened change stream over the channels named in a store config.
 * Construction is synchronous and network-free; the underlying stream opens
 * on the first {@link Streamer.demand} call.
 */
export interface Streamer {
  /**
   * Ensures the underlying stream is open, opening it on first call. Resolves
   * once streaming is live. Safe to call concurrently and repeatedly.
   */
  demand: () => Promise<void>;
  /** Closes the stream. A no-op when the stream was never demanded. */
  close: () => Promise<void>;
}

/**
 * Creates a lazy streamer that, once demanded, listens to configured channels
 * and invokes the appropriate listeners when data changes.
 *
 * @template ScopedStores - The type of the stores
 * @param params - Configuration for the streamer
 * @returns The lazy streamer handle
 */
export const createStreamer = <ScopedStores extends Stores>({
  openStreamer: streamOpener,
  storeConfig,
  handleError,
  store,
  onOpen,
  onReopen,
}: StreamerParams<ScopedStores>): Streamer => {
  let opened: Promise<ObservableStream> | null = null;
  const open = async (): Promise<ObservableStream> => {
    const configValues = Object.values(storeConfig);
    const channels = unique.unique(
      configValues.flatMap(({ listeners }) => listeners.map(({ channel }) => channel)),
    );
    const listenersForChannels: Record<
      string,
      ChannelListener<ScopedStores, z.ZodType>[]
    > = {};
    configValues.forEach(({ listeners }) =>
      listeners.forEach((lis) => {
        const { channel } = lis;
        listenersForChannels[channel] = [...(listenersForChannels[channel] || []), lis];
      }),
    );
    const stream = await streamOpener(channels, { onOpen, onReopen });
    const handleChange = (frame: framer.Frame) => {
      const namesInFrame = [...frame.uniqueNames];
      namesInFrame.sort(channelNameSort);
      void handleError(
        async () => {
          for (const name of namesInFrame) {
            const series = frame.get(name);
            const listeners = listenersForChannels[name];
            if (listeners == null) continue;
            for (const { onChange, schema } of listeners)
              await handleError(async () => {
                let parsed: z.output<typeof schema>[];
                if (!series.dataType.equals(DataType.JSON))
                  parsed = Array.from(series).map((s) => schema.parse(s));
                else parsed = series.parseJSON(schema);
                for (const changed of parsed)
                  await handleError(
                    () => onChange({ changed, store }),
                    `Failed to handle streamer change for ${name}`,
                    NotFoundError,
                  );
              }, `Failed to parse streamer change for ${name}`);
          }
        },
        `Failed to handle streamer change for ${strings.naturalLanguageJoin(namesInFrame)}`,
      );
    };
    stream.onChange(handleChange);
    return stream;
  };
  return {
    demand: async () => {
      opened ??= open();
      await opened;
    },
    close: async () => {
      if (opened == null) return;
      const streamer = await opened;
      await streamer.close();
    },
  };
};
