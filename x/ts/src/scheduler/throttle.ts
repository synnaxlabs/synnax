// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DebouncedFn } from "@/scheduler/debounce";
import { type CrudeTimeSpan, TimeSpan, TimeStamp } from "@/telem";

/** The function returned by throttle. It carries the same handles as a debounce. */
export type ThrottledFn<Args extends unknown[]> = DebouncedFn<Args>;

const noop = (): void => {};

/**
 * @returns a function that calls `func` with its latest arguments at most once per
 * `waitFor`. The first call after a quiet period runs on the next task, so every call
 * made in the current task joins it. A zero `waitFor` calls `func` synchronously.
 */
export const throttle = <Args extends unknown[]>(
  func: (...args: Args) => void,
  waitFor: CrudeTimeSpan,
): ThrottledFn<Args> => {
  const throttlePeriod = new TimeSpan(waitFor);
  if (throttlePeriod.valueOf() <= 0)
    return Object.assign((...args: Args) => func(...args), {
      cancel: noop,
      flush: noop,
    });
  let timeout: ReturnType<typeof setTimeout> | undefined;
  let lastInvokeTime: TimeStamp = TimeStamp.MIN;
  let latestArgs: Args | null = null;
  const invoke = (): void => {
    timeout = undefined;
    if (latestArgs === null) return;
    lastInvokeTime = TimeStamp.now();
    const args = latestArgs;
    latestArgs = null;
    func(...args);
  };
  const throttled = (...args: Args): void => {
    latestArgs = args;
    if (timeout !== undefined) return;
    const remaining = throttlePeriod.sub(TimeStamp.since(lastInvokeTime));
    // A clock that moved backwards leaves more than a full period remaining.
    const wait =
      remaining.valueOf() <= 0 || remaining.greaterThan(throttlePeriod)
        ? 0
        : remaining.milliseconds;
    timeout = setTimeout(invoke, wait);
  };
  const cancel = (): void => {
    clearTimeout(timeout);
    timeout = undefined;
    latestArgs = null;
  };
  const flush = (): void => {
    if (timeout === undefined) return;
    clearTimeout(timeout);
    invoke();
  };
  return Object.assign(throttled, { cancel, flush });
};
