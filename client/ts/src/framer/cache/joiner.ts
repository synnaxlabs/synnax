// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type breaker, type destructor, sync, type TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnsupportedError } from "@/errors";
import { type Joins, type Wait } from "@/framer/cache/join";
import { COMMIT_MARGIN, type Reader } from "@/framer/cache/reader";
import { NO_HISTORY, READ_FAILED } from "@/framer/cache/status";
import { type MultiplexedStreamer } from "@/framer/cache/streamer";
import { createBreaker, isDefinitive } from "@/framer/retry";
import { status } from "@/status";

export interface JoinerProps {
  joins: Joins;
  reader: Reader;
  streamer: MultiplexedStreamer;
  now: () => TimeStamp;
  /** Retry behavior for failed catch-up reads. */
  breaker?: breaker.Config;
}

// The catch-up reads of one index channel's join.
interface Driver {
  abort: AbortController;
  running: boolean;
  handlers: Set<() => void>;
  // breaker backs off failed reads, and landings that fail to join a member.
  breaker: breaker.Breaker;
  // landed holds the members that landed at least once.
  landed: Set<channel.Key>;
  // rejected holds the members a definitive failure stopped until they leave.
  rejected: Set<channel.Key>;
}

/**
 * Catches the joins of {@link Joins} up with their streams. After the stream acks a
 * waiting member and its index, a catch-up read fetches the member's open blocks, and
 * the member lands. A member the Core rejects stops with its error, and the rest of its
 * join keeps going.
 */
export class Joiner {
  private readonly drivers = new Map<channel.Key, Driver>();
  private readonly failures = new Map<channel.Key, status.Status>();
  private readonly notifier = new sync.Notifier();

  constructor(private readonly props: JoinerProps) {}

  /**
   * Joins the channel and its index until the returned destructor runs.
   * @param landed - Called after each catch-up read of the join lands.
   */
  join(ch: channel.Key, index: channel.Key, landed: () => void): destructor.Destructor {
    const { joins } = this.props;
    let driver = this.drivers.get(index);
    if (driver == null) {
      driver = {
        abort: new AbortController(),
        running: false,
        handlers: new Set(),
        breaker: createBreaker(this.notifier, this.props.breaker),
        landed: new Set(),
        rejected: new Set(),
      };
      this.drivers.set(index, driver);
    }
    driver.handlers.add(landed);
    joins.join(ch, index);
    this.wake([index]);
    let left = false;
    return () => {
      if (left) return;
      left = true;
      driver.handlers.delete(landed);
      joins.leave(ch, index);
      [ch, index].forEach((k) => {
        if (joins.has(k)) return;
        this.failures.delete(k);
        driver.landed.delete(k);
        driver.rejected.delete(k);
      });
      if (joins.has(index)) return;
      driver.abort.abort();
      this.drivers.delete(index);
    };
  }

  /** Runs the catch-up reads of the joins of the index channels. */
  wake(indexes: Iterable<channel.Key>): void {
    for (const index of indexes) void this.run(index);
  }

  /** @returns the status of the channel's last failed catch-up read, if any. */
  status(ch: channel.Key): status.Status | undefined {
    return this.failures.get(ch);
  }

  /** Stops every catch-up read. In-flight reads are dropped when they land. */
  close(): void {
    this.drivers.forEach((d) => d.abort.abort());
    this.drivers.clear();
    this.notifier.notify();
  }

  private async run(index: channel.Key): Promise<void> {
    const driver = this.drivers.get(index);
    if (driver == null || driver.running) return;
    driver.running = true;
    const { joins, reader, streamer, now } = this.props;
    const { signal } = driver.abort;
    let failed = false;
    try {
      for (;;) {
        const waits = joins.waiting(index).filter((w) => !driver.rejected.has(w.key));
        const keys = waits.map((w) => w.key);
        if (keys.length === 0 || signal.aborted) return;
        // A member that waits again before it placed a sample failed to join. One that
        // placed a sample shows that the join works.
        const back = keys.filter((k) => driver.landed.has(k));
        if (failed || back.some((k) => joins.bounced(k))) {
          if (!(await driver.breaker.wait()) || signal.aborted) return;
        } else if (back.length > 0) driver.breaker.reset();
        const results = await Promise.allSettled(
          keys.map(async (k) => {
            await streamer.waitAcked(k === index ? [k] : [k, index], signal);
            await reader.catchUp(k, joins.catchUpFrom(k, now().sub(COMMIT_MARGIN)));
          }),
        );
        if (signal.aborted) return;
        const caught: Wait[] = [];
        const historyless: Wait[] = [];
        failed = false;
        results.forEach((r, i) => {
          const k = keys[i];
          if (r.status === "fulfilled") {
            this.failures.delete(k);
            caught.push(waits[i]);
          } else if (UnsupportedError.matches(r.reason)) {
            // The Core rejects a read of a calculation over virtual channels, and of
            // its index, as unsupported. They store no samples, so they join without
            // history.
            this.failures.set(k, status.create(NO_HISTORY));
            historyless.push(waits[i]);
          } else {
            this.failures.set(k, status.fromException(r.reason, READ_FAILED));
            if (isDefinitive(r.reason)) driver.rejected.add(k);
            else failed = true;
          }
        });
        joins.landed(index, caught);
        joins.landed(index, historyless, true);
        [...caught, ...historyless].forEach(({ key }) => driver.landed.add(key));
        driver.handlers.forEach((f) => f());
      }
    } finally {
      driver.running = false;
    }
  }
}
