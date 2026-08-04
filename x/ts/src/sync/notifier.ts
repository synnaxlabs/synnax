// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@/telem";

/**
 * A sticky wake signal: waiters block until notified or until their span
 * elapses. Unlike a condition variable, a notify with no waiters is remembered
 * and released to the next {@link Notifier.wait}, so a wake can never be lost
 * to a caller that has not blocked yet.
 */
export class Notifier {
  private readonly waiters = new Set<() => void>();
  private pending = false;

  /**
   * Blocks until notified, or until the span elapses. Returns immediately when
   * a notify is already pending, consuming it.
   * @param span - How long to wait. Null waits indefinitely.
   */
  async wait(span: CrudeTimeSpan | null): Promise<void> {
    if (this.pending) {
      this.pending = false;
      return;
    }
    await new Promise<void>((resolve) => {
      let timer: ReturnType<typeof setTimeout> | undefined;
      const wake = (): void => {
        if (timer != null) clearTimeout(timer);
        this.waiters.delete(wake);
        resolve();
      };
      this.waiters.add(wake);
      if (span != null) timer = setTimeout(wake, new TimeSpan(span).milliseconds);
    });
  }

  /** Wakes every current waiter. With none, the next wait returns immediately. */
  notify(): void {
    if (this.waiters.size === 0) {
      this.pending = true;
      return;
    }
    // wake mutates the set, so iterate a copy
    [...this.waiters].forEach((wake) => wake());
  }
}
