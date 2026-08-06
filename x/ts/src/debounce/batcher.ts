// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@/telem/telem";

/** A request queued in a {@link Batcher} window, carrying the handles the executor
 *  uses to settle the caller's enqueue promise. */
export interface Entry<Req, Res = void> extends Pick<
  PromiseWithResolvers<Res>,
  "resolve" | "reject"
> {
  req: Req;
}

export interface BatcherProps<Req, Res = void> {
  /** Window for coalescing enqueues into one exec call. The window's timer is not
   * reset by later enqueues, so it is also the max wait before a batch fires. */
  interval: CrudeTimeSpan;
  /** Executes an accumulated batch, settling each entry. An error thrown or
   * rejected from it rejects the batch's unsettled entries. */
  exec: (entries: Array<Entry<Req, Res>>) => void | Promise<void>;
}

/**
 * Coalesces requests arriving within a fixed window into one executor call.
 * What a batch means is the executor's business: it receives every entry queued
 * during the window and settles each one.
 */
export class Batcher<Req, Res = void> {
  private readonly interval: TimeSpan;
  private readonly exec: BatcherProps<Req, Res>["exec"];
  private pending: Array<Entry<Req, Res>> | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;

  constructor({ interval, exec }: BatcherProps<Req, Res>) {
    this.interval = new TimeSpan(interval);
    this.exec = exec;
  }

  /**
   * Queues the request into the open window, opening one if none exists.
   * @returns A promise settled by the executor for this request.
   */
  enqueue(req: Req): Promise<Res> {
    const { promise, resolve, reject } = Promise.withResolvers<Res>();
    if (this.pending == null) {
      this.pending = [];
      this.timer = setTimeout(() => {
        const batch = this.pending;
        this.pending = null;
        this.timer = null;
        if (batch == null || batch.length === 0) return;
        void (async () => {
          try {
            await this.exec(batch);
          } catch (exc) {
            batch.forEach(({ reject }) => reject(exc));
          }
        })();
      }, this.interval.milliseconds);
    }
    this.pending.push({ req, resolve, reject });
    return promise;
  }

  /**
   * Rejects every queued entry with the reason and cancels the open window. The
   * batcher stays usable: a later enqueue opens a new window.
   */
  close(reason: unknown): void {
    if (this.timer != null) clearTimeout(this.timer);
    this.timer = null;
    const batch = this.pending;
    this.pending = null;
    batch?.forEach(({ reject }) => reject(reason));
  }
}
