// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { assert } from "vitest";

import { id } from "@/id";
import { sleep } from "@/sleep";
import { TimeSpan } from "@/telem";

export const toString = (value: unknown): string =>
  JSON.stringify(value, (_, value) => {
    if (typeof value === "bigint") return value.toString();
    return value;
  });

export const expectAlways = async (
  fn: () => void | Promise<void>,
  duration: number = 200,
  interval: number = 20,
) => {
  const start = Date.now();
  while (Date.now() - start < duration) {
    await fn();
    await sleep.sleep(TimeSpan.milliseconds(interval));
  }
};

/**
 * Throws when value is null or undefined, narrowing it for subsequent use.
 * Excludes void as well as null and undefined: NonNullable leaves `void & {}`
 * behind, which drops every property off a `T | void` return.
 */
export function assertDefined<T>(
  value: T,
  message = "expected value to be defined",
): asserts value is Exclude<T, null | undefined | void> {
  assert(value != null, message);
}

/**
 * Generates a unique, Core-safe resource name: letters, digits, and underscores only,
 * prefixed so failures are traceable back to tests.
 */
export const uniqueName = (prefix: string = "test"): string =>
  `${prefix}_${id.create()}`;
