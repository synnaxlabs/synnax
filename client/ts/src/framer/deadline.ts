// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import { errors, type TimeSpan } from "@synnaxlabs/x";

/**
 * @returns the value of the request, if it settles within the timeout.
 * @param message - Returns the message of the error thrown when the timeout passes.
 * @throws {Unreachable} if the timeout passes before the request settles.
 */
export const withDeadline = async <T>(
  request: Promise<T>,
  timeout: TimeSpan,
  message: () => string,
): Promise<T> => {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const deadline = new Promise<never>((_, reject) => {
    timer = setTimeout(
      () => reject(new Unreachable({ message: message() })),
      timeout.milliseconds,
    );
  });
  try {
    return await Promise.race([request, deadline]);
  } catch (err) {
    throw errors.fromUnknown(err);
  } finally {
    clearTimeout(timer);
  }
};
