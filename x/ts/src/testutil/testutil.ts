// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
    await new Promise((resolve) => setTimeout(resolve, interval));
  }
};
