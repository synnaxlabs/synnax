// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@/telem/telem";

// DebouncedFn is the function returned by debounce. It carries cancel and
// flush handles so callers (e.g., React cleanup paths) can drop pending
// invocations or force them to run immediately on demand. Shape matches
// lodash.debounce.
export interface DebouncedFn<Args extends unknown[]> {
  (...args: Args): void;
  cancel: () => void;
  flush: () => void;
}

const noop = (): void => {};

export const debounce = <Args extends unknown[]>(
  func: (...args: Args) => void,
  waitFor: CrudeTimeSpan,
): DebouncedFn<Args> => {
  const debouncePeriod = new TimeSpan(waitFor);
  if (debouncePeriod.valueOf() <= 0) {
    const passthrough = ((...args: Args) => func(...args)) as DebouncedFn<Args>;
    passthrough.cancel = noop;
    passthrough.flush = noop;
    return passthrough;
  }
  let timeout: ReturnType<typeof setTimeout> | undefined;
  let latestArgs: Args | null = null;
  const invoke = (): void => {
    if (latestArgs === null) return;
    const args = latestArgs;
    latestArgs = null;
    func(...args);
  };
  const debounced = ((...args: Args): void => {
    latestArgs = args;
    clearTimeout(timeout);
    timeout = setTimeout(() => {
      timeout = undefined;
      invoke();
    }, debouncePeriod.milliseconds);
  }) as DebouncedFn<Args>;
  debounced.cancel = (): void => {
    clearTimeout(timeout);
    timeout = undefined;
    latestArgs = null;
  };
  debounced.flush = (): void => {
    if (timeout === undefined) return;
    clearTimeout(timeout);
    timeout = undefined;
    invoke();
  };
  return debounced;
};
