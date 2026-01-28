// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const debounce = <F extends (...args: any[]) => void>(
  func: F,
  waitFor: number,
): F => {
  let timeout: ReturnType<typeof setTimeout> | null = null;
  if (waitFor === 0) return func;

  const debounced = (...args: Parameters<F>): void => {
    if (timeout !== null) {
      clearTimeout(timeout);
      timeout = null;
    }
    timeout = setTimeout(() => func(...args), waitFor);
  };

  return debounced as F;
};

export const throttle = <F extends (...args: unknown[]) => void>(
  func: F,
  waitFor: number,
): F => {
  let timeout: ReturnType<typeof setTimeout> | null = null;
  if (waitFor === 0) return func;

  const throttled = (...args: Parameters<F>): void => {
    if (timeout === null)
      timeout = setTimeout(() => {
        func(...args);
        timeout = null;
      }, waitFor);
  };

  return throttled as F;
};
