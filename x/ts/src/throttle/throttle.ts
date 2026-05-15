// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan, TimeStamp } from "@/telem/telem";

export const throttle = <Args extends unknown[]>(
  func: (...args: Args) => void,
  waitFor: CrudeTimeSpan,
): ((...args: Args) => void) => {
  const throttlePeriod = new TimeSpan(waitFor);
  if (throttlePeriod.valueOf() <= 0) return func;
  let timeout: NodeJS.Timeout | undefined;
  let lastInvokeTime: TimeStamp = TimeStamp.MIN;
  let latestArgs: Args | null = null;
  const invoke = (): void => {
    if (latestArgs === null) return;
    lastInvokeTime = TimeStamp.now();
    const args = latestArgs;
    latestArgs = null;
    func(...args);
  };
  return (...args: Args): void => {
    const timeSinceLastInvoke = TimeStamp.since(lastInvokeTime);
    latestArgs = args;
    const remaining = throttlePeriod.sub(timeSinceLastInvoke);
    if (remaining.valueOf() <= 0 || remaining.greaterThan(throttlePeriod)) {
      clearTimeout(timeout);
      timeout = undefined;
      invoke();
      return;
    }
    if (timeout != null) return;
    timeout = setTimeout(() => {
      timeout = undefined;
      invoke();
    }, remaining.milliseconds);
  };
};
