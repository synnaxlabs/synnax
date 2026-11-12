// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF } from "@synnaxlabs/freighter";
import { type breaker, caseconv, DataType, errors, zod } from "@synnaxlabs/x";
import type z from "zod";

import { isConnectionError, NotFoundError } from "@/errors";
import { type framer } from "@/framer";
import { HardenedStreamer, ObservableStreamer } from "@/framer/hardened";

/**
 * A raw channel reaction: parses each sample from the channel with the schema and
 * invokes onChange once per frame with every value that parses, in order. Mirror
 * listeners bind to this shape internally; domains register reactions via the cache's
 * listen.
 */
export interface Listener<Z extends z.ZodType = z.ZodType> {
  /** The name of the Synnax channel to listen to */
  channel: string;
  /** Zod schema for parsing and validating channel data */
  schema: Z;
  /** Receives the frame's parsed values for the channel as one batch. */
  onChange(this: void, changed: z.output<Z>[]): Promise<unknown> | unknown;
  /**
   * Receives each sample that fails the schema, as the schema received it, before
   * onChange gets the frame's valid values. The streamer reports the first error of
   * each frame too.
   */
  onInvalid?(this: void, raw: unknown, error: Error): void;
}

/**
 * Sorts channel names so deletions are processed before other changes, so a delete and
 * a re-create of one record in the same frame apply in order.
 * @returns Sort order (-1, 0, or 1)
 */
const channelNameSort = (a: string, b: string) => {
  const aHasDelete = a.includes("delete");
  const bHasDelete = b.includes("delete");
  if (aHasDelete && !bHasDelete) return -1;
  if (!aHasDelete && bHasDelete) return 1;
  return 0;
};

/** Arguments for opening a cache streamer. */
export interface StreamerParams {
  /** Receives frame-handling and listener errors. */
  onError: (error: Error) => void;
  /** The channel listeners to drive with streamed changes. */
  listeners: Listener[];
  /** Opens the raw frame stream; hardening is applied internally. */
  openStreamer: framer.StreamOpener;
  /** Retry behavior for stream reconnect attempts. */
  breaker?: breaker.Config;
  /** Called once when the stream first opens, before any frame is processed. */
  onOpen?: () => void;
  /**
   * Called after every successful reconnect of the underlying stream (not the
   * initial open). Changes may have been missed while disconnected.
   */
  onReopen?: () => void;
  /**
   * Called whenever the stream goes live: after the initial open and before every
   * onReopen, so reconciliation fetches never race a stale liveness verdict.
   */
  onLive?: () => void;
  /** Called when the underlying stream fails and reconnection begins. */
  onDrop?: (error: Error) => void;
  /** Receives each failed reconnect attempt. Defaults to console.warn. */
  onRetry?: (error: Error) => void;
  /**
   * Called when the stream ends on a failure it cannot recover from. The next
   * demand opens a new stream.
   */
  onDead?: (error: Error) => void;
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
 * Creates a lazy streamer that, once demanded, listens to configured channels and
 * invokes the appropriate listeners when data changes.
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
  onRetry,
  onDead,
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
  const deliver = async (fn: () => unknown, message: string) => {
    try {
      await fn();
    } catch (exc) {
      report(exc, message);
    }
  };
  const open = async (): Promise<ObservableStreamer> => {
    if (closed) throw new EOF();
    const listenersForChannels = Map.groupBy(allListeners, ({ channel }) => channel);
    const channels = Array.from(listenersForChannels.keys());
    const h = new HardenedStreamer(
      openStreamer,
      { channels },
      {
        breaker: breakerConfig,
        // onLive first: onReopen's reconcile fetches must not hit a stale liveness
        // short circuit
        onReopen: () => {
          onLive?.();
          onReopen?.();
        },
        onDrop,
        onRetry,
      },
    );
    // Held outside so close() can interrupt the retry loop while this call
    // is still waiting on a first successful open.
    hardened = h;
    await h.start();
    // ObservableStreamer construction below starts reads, so onOpen precedes any frame.
    onOpen?.();
    onLive?.();
    // A listener can await, so each frame waits for the one before it to apply.
    let applied: Promise<void> = Promise.resolve();
    const handleChange = (frame: framer.Frame) => {
      const namesInFrame = frame.uniqueNames.sort(channelNameSort);
      applied = applied.then(async () => {
        for (const name of namesInFrame) {
          const series = frame.get(name);
          const listeners = listenersForChannels.get(name);
          if (listeners == null) continue;
          const json = series.dataType.equals(DataType.JSON);
          const samples: unknown[] = json ? series.toStrings() : Array.from(series);
          const parseOpts = { label: `${name} sample` };
          for (const { onChange, onInvalid, schema } of listeners) {
            const parsed: z.output<typeof schema>[] = [];
            let reported = false;
            for (const sample of samples) {
              let raw = sample;
              try {
                // Decoded here so onInvalid gets the value that the schema rejected.
                if (json)
                  raw = caseconv.snakeToCamel(JSON.parse(sample as string), { schema });
                parsed.push(zod.parse(schema, raw, parseOpts));
              } catch (exc) {
                // One report per frame, so a frame of bad samples cannot flood onError.
                if (!reported)
                  report(exc, `failed to parse streamer change for ${name}`);
                reported = true;
                if (onInvalid != null)
                  await deliver(
                    () => onInvalid(raw, errors.fromUnknown(exc)),
                    `failed to handle invalid streamer change for ${name}`,
                  );
              }
            }
            if (parsed.length > 0)
              await deliver(
                () => onChange(parsed),
                `failed to handle streamer change for ${name}`,
              );
          }
        }
      });
    };
    const stream = new ObservableStreamer(h, undefined, (error) => {
      // drop the memoized stream so the next demand opens a fresh one
      opened = null;
      onDead?.(error);
    });
    stream.onChange(handleChange);
    return stream;
  };
  return {
    demand: async () => {
      if (opened == null) {
        const { promise, resolve, reject } =
          Promise.withResolvers<ObservableStreamer>();
        // Assigned before open runs: opening resolves channels through the cached
        // read path, which re-enters demand synchronously. An unset memo there
        // recurses instead of joining. Keep these two statements in this order.
        opened = promise;
        open().then(resolve, reject);
      }
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
