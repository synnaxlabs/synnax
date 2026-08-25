// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@/telem";

/**
 * Bounds how long the caller waits on promise.
 * @param span - The deadline. A non-positive span waits forever.
 * @param onTimeout - Builds the error to reject with when the deadline passes.
 * @returns the promise's value.
 * @throws the promise's own rejection, or the error from onTimeout.
 */
export const withTimeout = async <V>(
  promise: Promise<V>,
  span: TimeSpan,
  onTimeout: () => Error,
): Promise<V> => {
  if (span.lessThanOrEqual(TimeSpan.ZERO)) return await promise;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const deadline = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(onTimeout()), span.milliseconds);
  });
  try {
    return await Promise.race([promise, deadline]);
  } finally {
    clearTimeout(timer);
    // A promise that loses the race still settles. Left alone, a late rejection
    // reaches nobody and surfaces as an unhandled one.
    promise.catch(() => {});
  }
};
