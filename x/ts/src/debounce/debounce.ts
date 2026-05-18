// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@/telem/telem";

export const debounce = <Args extends unknown[]>(
  func: (...args: Args) => void,
  waitFor: CrudeTimeSpan,
): ((...args: Args) => void) => {
  const debouncePeriod = new TimeSpan(waitFor);
  if (debouncePeriod.valueOf() <= 0) return func;
  let timeout: NodeJS.Timeout | undefined;
  let latestArgs: Args | null = null;
  const invoke = (): void => {
    if (latestArgs === null) return;
    const args = latestArgs;
    latestArgs = null;
    func(...args);
  };
  return (...args: Args): void => {
    latestArgs = args;
    clearTimeout(timeout);
    timeout = setTimeout(() => {
      timeout = undefined;
      invoke();
    }, debouncePeriod.milliseconds);
  };
};
