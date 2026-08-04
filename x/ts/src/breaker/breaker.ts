// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { sleep } from "@/sleep";
import { type CrudeTimeSpan, TimeSpan } from "@/telem";

/** Paces a retry loop on capped, optionally jittered exponential backoff. */
export class Breaker {
  private readonly config: Required<z.infer<typeof breakerConfigZ>> & {
    sleepFn: (duration: TimeSpan) => Promise<void>;
  };
  private retries: number;
  private interval: TimeSpan;

  constructor(cfg?: Config) {
    this.config = {
      baseInterval: new TimeSpan(cfg?.baseInterval ?? TimeSpan.seconds(1)),
      maxInterval: new TimeSpan(cfg?.maxInterval ?? TimeSpan.MAX),
      maxRetries: cfg?.maxRetries ?? 5,
      scale: cfg?.scale ?? 1,
      jitter: cfg?.jitter ?? 0,
      sleepFn: cfg?.sleepFn ?? sleep.sleep,
    };
    this.retries = 0;
    this.interval = new TimeSpan(this.config.baseInterval);
  }

  /**
   * Sleeps the current retry interval, then scales it for the next call.
   * Returns false without sleeping once `maxRetries` is exhausted.
   */
  async wait(): Promise<boolean> {
    const { maxRetries, maxInterval, scale, jitter, sleepFn } = this.config;
    if (this.retries >= maxRetries) return false;
    let interval = this.interval;
    if (jitter > 0)
      interval = TimeSpan.milliseconds(
        interval.milliseconds * (1 + (Math.random() * 2 - 1) * jitter),
      );
    await sleepFn(interval);
    this.interval = this.interval.mult(scale);
    if (this.interval.greaterThan(maxInterval)) this.interval = maxInterval;
    this.retries++;
    return true;
  }

  /** A log-ready summary of the retry count and the next interval. */
  get retryMessage(): string {
    return `breaker triggered ${this.retries + 1}/${this.config.maxRetries} times, retrying in ${this.interval.toString()}`;
  }

  /** Returns the retry count and interval to their starting values. */
  reset() {
    this.retries = 0;
    this.interval = this.config.baseInterval;
  }
}

/** Schema for the serializable fields of {@link Config}. */
export const breakerConfigZ = z.object({
  baseInterval: TimeSpan.z.optional(),
  maxInterval: TimeSpan.z.optional(),
  maxRetries: z.number().optional(),
  scale: z.number().optional(),
  jitter: z.number().min(0).max(1).optional(),
});

/** Configures a {@link Breaker}. */
export interface Config extends z.input<typeof breakerConfigZ> {
  /** Duration of the first wait. Defaults to 1 second. */
  baseInterval?: CrudeTimeSpan;
  /** Ceiling on the scaled retry interval. Defaults to unbounded. */
  maxInterval?: CrudeTimeSpan;
  /** Waits allowed before `wait` returns false. Defaults to 5. */
  maxRetries?: number;
  /** Multiplier applied to the interval after each wait. Defaults to 1. */
  scale?: number;
  /**
   * Random fraction added to or subtracted from each wait to desynchronize
   * concurrent retries. Defaults to 0.
   */
  jitter?: number;
  /** Performs the wait. Defaults to a plain sleep. */
  sleepFn?: (duration: TimeSpan) => Promise<void>;
}
