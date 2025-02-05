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

export const breakerConfig = z.object({
  interval: TimeSpan.z.optional(),
  maxRetries: z.number().optional(),
  scale: z.number().optional(),
});

export interface Config extends Omit<z.infer<typeof breakerConfig>, "interval"> {
  interval?: CrudeTimeSpan;
  maxRetries?: number;
  scale?: number;
  sleepFn?: (duration: TimeSpan) => Promise<void>;
}

export const create = (options: Config = {}): (() => Promise<boolean>) => {
  const sleepFn = options.sleepFn || sleep.sleep;
  const maxRetries = options.maxRetries ?? 5;
  const scale = options.scale ?? 1;
  let retries = 0;
  let interval = new TimeSpan(options.interval ?? TimeSpan.milliseconds(1));
  return async () => {
    // Change from arrow function to regular function to preserve 'this'
    if (retries >= maxRetries) return false;
    await sleepFn(interval);
    interval = interval.mult(scale);
    retries++;
    return true;
  };
};
