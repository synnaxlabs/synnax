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
 * @returns the value of the promise, if it settles within the timeout. The timeout does
 * not cancel the promise, so a caller that holds a resource for it releases the resource
 * in a `finally`.
 * @param error - Returns the error thrown when the timeout passes.
 * @throws the error that `error` returns, if the timeout passes before the promise
 * settles.
 */
export const withTimeout = async <T>(
  promise: Promise<T>,
  timeout: CrudeTimeSpan,
  error: () => Error | Promise<Error>,
): Promise<T> => {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const expired = new Promise<never>((_, reject) => {
    timer = setTimeout(
      () => void Promise.resolve().then(error).then(reject, reject),
      new TimeSpan(timeout).milliseconds,
    );
  });
  try {
    return await Promise.race([promise, expired]);
  } finally {
    clearTimeout(timer);
  }
};
