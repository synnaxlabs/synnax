// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep } from "@/sleep";
import { type CrudeTimeSpan, TimeSpan } from "@/telem";

export const toString = (value: unknown): string =>
  JSON.stringify(value, (_, value) => {
    if (typeof value === "bigint") return value.toString();
    return value;
  });

/**
 * Calls fn once per interval until the duration elapses. A bare number is milliseconds.
 * @throws the first error that fn throws.
 */
export const expectAlways = async (
  fn: () => void | Promise<void>,
  duration: CrudeTimeSpan = TimeSpan.milliseconds(200),
  interval: CrudeTimeSpan = TimeSpan.milliseconds(20),
): Promise<void> => {
  const span = TimeSpan.fromMilliseconds(duration).milliseconds;
  const start = Date.now();
  while (Date.now() - start < span) {
    await fn();
    await sleep.sleep(interval);
  }
};
