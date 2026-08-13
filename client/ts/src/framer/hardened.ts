// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The query cache value-imports this module, so a runtime edge back into
// channel or ontology re-creates the import cycle this file exists to break.
// Runtime imports must stay on leaf modules (external packages, @/errors).

import { EOF } from "@synnaxlabs/freighter";
import { breaker, errors, observe, sync, TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { AccessDeniedError, NotFoundError, ValidationError } from "@/errors";
import { type Frame } from "@/framer/frame";
import {
  type Streamer,
  type StreamerConfig,
  type StreamOpener,
} from "@/framer/streamer";

type NormalizedStreamerConfig = Extract<StreamerConfig, { channels: unknown }>;

// Copies object configs: update() reassigns channels, and the caller's config must
// not observe that.
const normalizeConfig = (config: StreamerConfig): NormalizedStreamerConfig =>
  typeof config === "object" && !Array.isArray(config) && "channels" in config
    ? { ...config }
    : { channels: config };

/**
 * A hardened streamer that automatically reconnects on failure.
 * This streamer wraps a regular streamer and adds automatic reconnection
 * logic when the connection is lost or errors occur.
 */
export class HardenedStreamer implements Streamer {
  private current: Streamer | null = null;
  private readonly breaker: breaker.Breaker;
  private readonly opener: StreamOpener;
  private readonly config: NormalizedStreamerConfig;
  private readonly onReopen?: () => void;
  private readonly onDrop?: (error: Error) => void;
  private readonly closeNotifier = new sync.Notifier();
  // A stream that outlives this span proves the incident over; a shorter
  // life keeps the reconnect backoff escalating.
  private readonly stableAfter: TimeSpan;
  private openedAt = TimeStamp.now();
  private closed = false;

  constructor(
    opener: StreamOpener,
    config: StreamerConfig,
    breakerConfig: breaker.Config = {},
    onReopen?: () => void,
    onDrop?: (error: Error) => void,
  ) {
    this.opener = opener;
    this.config = normalizeConfig(config);
    const {
      maxRetries = Infinity,
      baseInterval = TimeSpan.seconds(1),
      // reconnect attempts are cheap: a low cap bounds how long a returned
      // cluster goes unnoticed while backoff is escalated
      maxInterval = TimeSpan.seconds(5),
      scale = 2,
      jitter = 0.25,
    } = breakerConfig ?? {};
    this.breaker = new breaker.Breaker({
      maxRetries,
      baseInterval,
      maxInterval,
      scale,
      jitter,
      // close() interrupts a pending backoff so a retired streamer's
      // reconnect loop ends promptly instead of sleeping through it
      sleepFn: async (duration) => {
        await this.closeNotifier.wait(duration);
      },
    });
    this.stableAfter = new TimeSpan(maxInterval);
    this.onReopen = onReopen;
    this.onDrop = onDrop;
  }

  /**
   * Opens a new hardened streamer with the given configuration.
   * @param opener - The function to use for opening streamers
   * @param config - The configuration for the streamer
   * @param breakerConfig - Retry behavior for reconnect attempts. Defaults to
   * retrying forever on capped, jittered exponential backoff.
   * @param onReopen - Called after every successful reconnect (not the initial
   * open). Frames may have been dropped between the failure and the reopen.
   * @param onDrop - Called when the stream fails and reconnection begins.
   * @returns A promise that resolves to a new hardened streamer
   */
  static async open(
    opener: StreamOpener,
    config: StreamerConfig,
    breakerConfig?: breaker.Config,
    onReopen?: () => void,
    onDrop?: (error: Error) => void,
  ): Promise<HardenedStreamer> {
    const h = new HardenedStreamer(opener, config, breakerConfig, onReopen, onDrop);
    await h.start();
    return h;
  }

  /**
   * Opens the underlying stream, retrying connectivity failures under the breaker.
   * Call once after construction; {@link open} does both.
   * @throws the open failure when it is not retriable or the breaker is exhausted.
   */
  async start(): Promise<void> {
    await this.runStreamer(false);
  }

  private async runStreamer(notifyReopen: boolean = true): Promise<void> {
    // A long-lived stream that dropped is a fresh incident: reconnect
    // immediately. A short-lived one is a failing cycle (e.g. a server that
    // accepts streams and instantly ends them): keep backing off, or the
    // loop dials hot.
    if (notifyReopen)
      if (TimeStamp.since(this.openedAt).greaterThan(this.stableAfter))
        this.breaker.reset();
      else if (!(await this.breaker.wait())) throw new EOF();
    while (!this.closed)
      try {
        if (this.current != null) this.current.close();
        this.current = null;
        const next = await this.opener(this.config);
        // closed while the open was in flight: the fresh stream is already
        // retired and must not be handed to callers
        if (this.closed) {
          next.close();
          break;
        }
        this.current = next;
        this.openedAt = TimeStamp.now();
        if (notifyReopen) this.onReopen?.();
        return;
      } catch (e) {
        this.current = null;
        if (this.closed) break;
        const err = errors.fromUnknown(e);
        // Retrying only fixes connectivity; a definitive rejection recurs on every
        // attempt. Expired and invalid tokens are not definitive: the auth
        // middleware refreshes them, and a later attempt gets a fresh budget.
        if (
          AccessDeniedError.matches(err) ||
          ValidationError.matches(err) ||
          NotFoundError.matches(err)
        )
          throw err;
        if (!(await this.breaker.wait())) throw err;
        // close() interrupts the backoff above, so re-check: a retired
        // streamer's failed reconnect is expected, not a fault to report.
        if (this.closed) break;
        console.error("failed to open streamer", e);
      }
    throw new EOF();
  }

  private get wrapped(): Streamer {
    if (this.current == null) throw new Error("stream closed");
    return this.current;
  }

  async update(channels: channel.Params): Promise<void> {
    if (this.closed) throw new EOF();
    this.config.channels = channels;
    try {
      await this.wrapped.update(channels);
    } catch (e) {
      if (this.closed || EOF.matches(e)) throw errors.fromUnknown(e);
      this.onDrop?.(errors.fromUnknown(e));
      await this.runStreamer();
      return await this.update(channels);
    }
  }

  async next(): Promise<IteratorResult<Frame>> {
    try {
      return { done: false, value: await this.read() };
    } catch (e) {
      if (EOF.matches(e)) return { done: true, value: undefined };
      throw errors.fromUnknown(e);
    }
  }

  async read(): Promise<Frame> {
    if (this.closed) throw new EOF();
    try {
      return await this.wrapped.read();
    } catch (e) {
      if (this.closed) throw new EOF();
      // an EOF the client did not ask for is a drop: the server ended the
      // stream, so reconnect like any other failure
      this.onDrop?.(errors.fromUnknown(e));
      await this.runStreamer();
      return await this.read();
    }
  }

  /** Stops reconnecting and closes the stream. Idempotent; never throws. */
  close(): void {
    if (this.closed) return;
    this.closed = true;
    this.closeNotifier.notify();
    this.current?.close();
  }

  get keys(): channel.Key[] {
    return this.wrapped.keys;
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame> {
    return this;
  }
}

/**
 * Wraps a standard streamer to implement an observable interface for handling changes
 * to channel values through an onChange handler.
 */
export class ObservableStreamer<V = Frame>
  extends observe.Observer<Frame, V>
  implements observe.ObservableAsyncCloseable<V>
{
  private readonly streamer: Streamer;
  private readonly closePromise: Promise<void>;
  private readonly onDead?: (error: Error) => void;
  private closeRequested = false;

  /**
   * Creates a new observable streamer.
   * @param streamer - The streamer to wrap
   * @param transform - An optional transform function to apply to each frame
   * @param onDead - Called when the stream ends on a failure the wrapped streamer
   * will not recover from. No frames follow.
   * @template V - The type of the transformed value. Only relevant if transform is
   * provided. Defaults to Frame.
   */
  constructor(
    streamer: Streamer,
    transform?: observe.Transform<Frame, V>,
    onDead?: (error: Error) => void,
  ) {
    super(transform);
    this.streamer = streamer;
    this.onDead = onDead;
    // Nothing awaits closePromise until close(), so a failure would go unhandled.
    this.closePromise = this.stream().catch(console.error);
  }

  async update(channels: channel.Params): Promise<void> {
    await this.streamer.update(channels);
  }

  async close(): Promise<void> {
    this.closeRequested = true;
    this.streamer.close();
    await this.closePromise;
  }

  private async stream(): Promise<void> {
    try {
      for await (const frame of this.streamer) this.notify(frame);
    } catch (exc) {
      this.onDead?.(errors.fromUnknown(exc));
      return;
    }
    if (!this.closeRequested) this.onDead?.(new EOF());
  }
}
