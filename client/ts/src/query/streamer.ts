// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF } from "@synnaxlabs/freighter";
import { type breaker, DataType, errors, unique } from "@synnaxlabs/x";
import type z from "zod";

import { isConnectionError, NotFoundError } from "@/errors";
import { type framer } from "@/framer";
import { HardenedStreamer, ObservableStreamer } from "@/framer/hardened";

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
 * Arguments for opening a cache streamer.
 */
export interface StreamerParams {
  /** Receives frame-handling and listener errors. */
  onError: (error: Error) => void;
  /** The channel listeners to drive with streamed changes. */
  listeners: Listener[];
  /** Opens the raw frame stream; hardening is applied internally. */
  openStreamer: framer.StreamOpener;
  /** Retry behavior for stream reconnect attempts. */
  breaker?: breaker.Config;
  /**
   * Called once when the stream first opens, before any frame is processed.
   */
  onOpen?: () => void;
  /**
   * Called after every successful reconnect of the underlying stream (not the
   * initial open). Changes may have been missed while disconnected.
   */
  onReopen?: () => void;
  /**
   * Called whenever the stream goes live: after the initial open and before
   * every onReopen, so reconciliation fetches never race a stale liveness
   * verdict.
   */
  onLive?: () => void;
  /** Called when the underlying stream fails and reconnection begins. */
  onDrop?: (error: Error) => void;
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
  openStreamer,
  breaker: breakerConfig,
  listeners: allListeners,
  onError,
  onOpen,
  onReopen,
  onLive,
  onDrop,
}: StreamerParams): Streamer => {
  let opened: Promise<ObservableStreamer> | null = null;
  let hardened: HardenedStreamer | null = null;
  let closed = false;
  const report = (exc: unknown, message: string) => {
    if (NotFoundError.matches(exc)) return;
    // A connectivity failure mid-change is repaired by the reconcile that
    // follows the stream's reopen, so it is churn, not a defect.
    if (isConnectionError(exc)) return;
    onError(new Error(message, { cause: exc }));
  };
  const open = async (): Promise<ObservableStreamer> => {
    if (closed) throw new EOF();
    const channels = unique.unique(allListeners.map(({ channel }) => channel));
    const listenersForChannels: Record<string, Listener<z.ZodType>[]> = {};
    allListeners.forEach((lis) => {
      const { channel } = lis;
      listenersForChannels[channel] = [...(listenersForChannels[channel] || []), lis];
    });
    const h = new HardenedStreamer(
      openStreamer,
      { channels },
      breakerConfig,
      // onLive first: onReopen's reconcile fetches must not hit a stale
      // liveness short circuit
      () => {
        onLive?.();
        onReopen?.();
      },
      (error) => onDrop?.(error),
    );
    // Held outside so close() can interrupt the retry loop while this call
    // is still waiting on a first successful open.
    hardened = h;
    await h.start();
    // Reads start when the ObservableStreamer is constructed below, so onOpen
    // fires strictly before any frame or reconnect.
    onOpen?.();
    onLive?.();
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
    const stream = new ObservableStreamer(h);
    stream.onChange(handleChange);
    return stream;
  };
  return {
    demand: async () => {
      // The memo must be assigned before open's body runs: opening resolves
      // channels through the cached read path, which re-enters demand
      // synchronously. An unset memo there recurses instead of joining.
      opened ??= Promise.resolve().then(open);
      try {
        await opened;
      } catch (exc) {
        // clear the memoized failure so a later demand can retry the open
        opened = null;
        // a close mid-open retires the stream; the demand did not fail
        if (closed) return;
        throw errors.fromUnknown(exc);
      }
    },
    close: async () => {
      closed = true;
      // Interrupts an in-flight open's retry loop; opened then settles.
      hardened?.close();
      if (opened == null) return;
      const streamer = await opened.catch(() => null);
      if (streamer != null) await streamer.close();
    },
  };
};
