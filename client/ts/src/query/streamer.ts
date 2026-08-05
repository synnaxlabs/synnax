// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, errors, unique } from "@synnaxlabs/x";
import type z from "zod";

import { isConnectionError, NotFoundError } from "@/errors";
import { type framer } from "@/framer";

/**
 * A raw channel reaction: parses frames from the channel with the schema and
 * invokes onChange per parsed value. Mirror listeners bind to this shape
 * internally; domains register reactions via the cache's listen.
 */
export interface Listener<Z extends z.ZodType = z.ZodType> {
  /** The name of the Synnax channel to listen to */
  channel: string;
  /** Zod schema for parsing and validating channel data */
  schema: Z;
  /** Callback function invoked when the channel data changes */
  onChange(this: void, changed: z.output<Z>): Promise<unknown> | unknown;
}

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

/**
 * An opened change stream: frame notifications plus close.
 *
 * TODO: exists only because a concrete framer.ObservableStreamer would close
 * the import cycle query -> framer -> channel -> query. Re-architect the
 * cache to remove it.
 */
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
 *
 * TODO: exists for the same import cycle as {@link ObservableStream}; remove
 * both when the cache is re-architected.
 */
export interface StreamOpener {
  (channels: string[], hooks: StreamOpenerHooks): Promise<ObservableStream>;
}

/**
 * Arguments for opening a cache streamer.
 */
export interface StreamerParams {
  /** Receives frame-handling and listener errors. */
  onError: (error: Error) => void;
  /** The channel listeners to drive with streamed changes. */
  listeners: Listener[];
  /** Function to open the change stream */
  openStreamer: StreamOpener;
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
 * A lazily-opened change stream over the channels named in a table config.
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
 * @param params - Configuration for the streamer
 * @returns The lazy streamer handle
 */
export const createStreamer = ({
  openStreamer: streamOpener,
  listeners: allListeners,
  onError,
  onOpen,
  onReopen,
}: StreamerParams): Streamer => {
  let opened: Promise<ObservableStream> | null = null;
  const report = (exc: unknown, message: string) => {
    if (NotFoundError.matches(exc)) return;
    // A connectivity failure mid-change is repaired by the reconcile that
    // follows the stream's reopen, so it is churn, not a defect.
    if (isConnectionError(exc)) return;
    onError(new Error(message, { cause: exc }));
  };
  const open = async (): Promise<ObservableStream> => {
    const channels = unique.unique(allListeners.map(({ channel }) => channel));
    const listenersForChannels: Record<string, Listener<z.ZodType>[]> = {};
    allListeners.forEach((lis) => {
      const { channel } = lis;
      listenersForChannels[channel] = [...(listenersForChannels[channel] || []), lis];
    });
    const stream = await streamOpener(channels, { onOpen, onReopen });
    const handleChange = (frame: framer.Frame) => {
      const namesInFrame = [...frame.uniqueNames];
      namesInFrame.sort(channelNameSort);
      void (async () => {
        for (const name of namesInFrame) {
          const series = frame.get(name);
          const listeners = listenersForChannels[name];
          if (listeners == null) continue;
          for (const { onChange, schema } of listeners) {
            let parsed: z.output<typeof schema>[];
            try {
              if (!series.dataType.equals(DataType.JSON))
                parsed = Array.from(series).map((s) => schema.parse(s));
              else parsed = series.parseJSON(schema);
            } catch (exc) {
              report(exc, `failed to parse streamer change for ${name}`);
              continue;
            }
            for (const changed of parsed)
              try {
                await onChange(changed);
              } catch (exc) {
                report(exc, `failed to handle streamer change for ${name}`);
              }
          }
        }
      })();
    };
    stream.onChange(handleChange);
    return stream;
  };
  return {
    demand: async () => {
      opened ??= open();
      try {
        await opened;
      } catch (exc) {
        // clear the memoized failure so a later demand can retry the open
        opened = null;
        throw errors.fromUnknown(exc);
      }
    },
    close: async () => {
      if (opened == null) return;
      const streamer = await opened;
      await streamer.close();
    },
  };
};
