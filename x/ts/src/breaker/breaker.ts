// Copyright 2025 Synnax Labs, Inc.
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

export class Breaker {
  private readonly config: Required<Config>;
  private retries: number;
  private interval: TimeSpan;

  constructor(cfg?: Config) {
    this.config = {
      baseInterval: cfg?.baseInterval ?? TimeSpan.seconds(1),
      maxRetries: cfg?.maxRetries ?? 5,
      scale: cfg?.scale ?? 1,
      sleepFn: cfg?.sleepFn ?? sleep.sleep,
    };
    this.retries = 0;
    this.interval = new TimeSpan(this.config.baseInterval);
  }

  async wait(): Promise<boolean> {
    const { maxRetries, scale, sleepFn } = this.config;
    if (this.retries >= maxRetries) return false;
    await sleepFn(this.interval);
    this.interval = this.interval.mult(scale);
    this.retries++;
    return true;
  }

  reset() {
    this.retries = 0;
    this.interval = new TimeSpan(this.config.baseInterval);
  }
}

export const breakerConfig = z.object({
  baseInterval: TimeSpan.z.optional(),
  maxRetries: z.number().optional(),
  scale: z.number().optional(),
});

export interface Config extends Omit<z.infer<typeof breakerConfig>, "baseInterval"> {
  baseInterval?: CrudeTimeSpan;
  maxRetries?: number;
  scale?: number;
  sleepFn?: (duration: TimeSpan) => Promise<void>;
}
