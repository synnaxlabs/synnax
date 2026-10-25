// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import {
  breaker,
  type destructor,
  sync,
  TimeSpan,
  type TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { AccessDeniedError, NotFoundError, ValidationError } from "@/errors";
import { rawIndex } from "@/framer/cache/block";
import { COMMIT_MARGIN, type Reader } from "@/framer/cache/reader";
import { type Store } from "@/framer/cache/store";
import { type MultiplexedStreamer } from "@/framer/cache/streamer";
import { status } from "@/status";

export interface JoinerProps {
  store: Store;
  reader: Reader;
  streamer: MultiplexedStreamer;
  now: () => TimeStamp;
  /** Retry behavior for failed catch-up reads. */
  breaker?: breaker.Config;
}

/** @returns true for a failure that recurs on every retry. */
export const isDefinitive = (e: unknown): boolean =>
  AccessDeniedError.matches(e) ||
  ValidationError.matches(e) ||
  NotFoundError.matches(e);

const NO_HISTORY: status.Crude = {
  variant: "warning",
  message: "Failed to read calculated channel history",
  description: "The channel shows only the samples streamed since it opened.",
};

// One index channel's join and the catch-up reads that drive it.
interface Driver {
  abort: AbortController;
  running: boolean;
  landed: Set<() => void>;
}

/**
 * Joins the streams of channels to their raw blocks. After the stream acks a joined
 * channel and its index, a catch-up read fetches their open blocks, and the store then
 * writes their streamed samples into the blocks. A catch-up read runs again whenever
 * the join loses its place, as after a gap in the stream.
 */
export class Joiner {
  private readonly props: Required<Omit<JoinerProps, "breaker">> & {
    breaker: breaker.Config;
  };
  private readonly drivers = new Map<channel.Key, Driver>();
  private readonly failures = new Map<channel.Key, status.Status>();
  private readonly notifier = new sync.Notifier();
  private readonly stopWaits: destructor.Destructor;
  private closed = false;

  constructor({ breaker: brk = {}, ...rest }: JoinerProps) {
    this.props = { ...rest, breaker: brk };
    this.stopWaits = rest.store.onWait((index) => void this.run(index));
  }

  /**
   * Joins the channel and its index until the returned destructor runs.
   * @param landed - Called after each catch-up read of the join lands.
   */
  join(ch: channel.Key, index: channel.Key, landed: () => void): destructor.Destructor {
    let driver = this.drivers.get(index);
    if (driver == null) {
      driver = { abort: new AbortController(), running: false, landed: new Set() };
      this.drivers.set(index, driver);
    }
    driver.landed.add(landed);
    this.props.store.join(ch, index);
    let left = false;
    return () => {
      if (left) return;
      left = true;
      driver.landed.delete(landed);
      this.props.store.leave(ch, index);
      if (this.props.store.members(index).length > 0) return;
      driver.abort.abort();
      this.drivers.delete(index);
      this.failures.delete(ch);
      this.failures.delete(index);
    };
  }

  /** @returns the status of the channel's last failed catch-up read, if any. */
  status(ch: channel.Key): status.Status | undefined {
    return this.failures.get(ch);
  }

  /** Stops every catch-up read. In-flight reads are dropped when they land. */
  close(): void {
    this.closed = true;
    this.stopWaits();
    this.drivers.forEach((d) => d.abort.abort());
    this.drivers.clear();
    this.notifier.notify();
  }

  private async run(index: channel.Key): Promise<void> {
    const driver = this.drivers.get(index);
    if (driver == null || driver.running) return;
    driver.running = true;
    const { store, reader, streamer, now } = this.props;
    const brk = new breaker.Breaker({
      baseInterval: TimeSpan.seconds(1),
      maxInterval: TimeSpan.seconds(30),
      maxRetries: Infinity,
      scale: 2,
      jitter: 0.25,
      sleepFn: async (span) => await this.notifier.wait(span),
      ...this.props.breaker,
    });
    const { signal } = driver.abort;
    try {
      for (;;) {
        const keys = store.waiting(index);
        if (keys.length === 0 || signal.aborted) return;
        try {
          await streamer.waitAcked(store.members(index), signal);
          const at = now().sub(COMMIT_MARGIN);
          const last = rawIndex(now());
          await Promise.all(
            keys.map(
              async (k) => await reader.catchUp(k, store.catchUpFrom(k, at), last),
            ),
          );
          if (signal.aborted) return;
          store.landed(index, keys);
          keys.forEach((k) => this.failures.delete(k));
          brk.reset();
        } catch (e) {
          if (signal.aborted || this.closed) return;
          // A calculation over virtual channels has no history, so it joins without it.
          const historyless = keys.filter(
            (k) => store.isCalculated(k) && !Unreachable.matches(e),
          );
          if (historyless.length > 0) {
            store.landed(index, historyless, true);
            historyless.forEach((k) => this.failures.set(k, status.create(NO_HISTORY)));
          }
          keys
            .filter((k) => !historyless.includes(k))
            .forEach((k) =>
              this.failures.set(
                k,
                status.fromException(e, "Failed to read channel data"),
              ),
            );
          driver.landed.forEach((f) => f());
          if (historyless.length === keys.length) continue;
          if (isDefinitive(e) || !(await brk.wait())) return;
          continue;
        }
        driver.landed.forEach((f) => f());
      }
    } finally {
      driver.running = false;
    }
  }
}
