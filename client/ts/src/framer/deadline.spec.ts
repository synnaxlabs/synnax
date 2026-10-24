// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import { TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { withDeadline } from "@/framer/deadline";

describe("withDeadline", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should return the value of a request that settles in time", async () => {
    const result = withDeadline(Promise.resolve(1), TimeSpan.seconds(1), () => "late");
    expect(await result).toEqual(1);
    expect(vi.getTimerCount()).toEqual(0);
  });

  it("should throw the error of a request that fails in time", async () => {
    const failed = withDeadline(
      Promise.reject(new Error("fetch exploded")),
      TimeSpan.seconds(1),
      () => "late",
    );
    await expect(failed).rejects.toThrow("fetch exploded");
    expect(vi.getTimerCount()).toEqual(0);
  });

  it("should throw Unreachable when the timeout passes first", async () => {
    const request = Promise.withResolvers<number>();
    const late = withDeadline(request.promise, TimeSpan.seconds(1), () => "too slow");
    const settled = expect(late).rejects.toThrow(
      new Unreachable({ message: "too slow" }),
    );
    await vi.advanceTimersByTimeAsync(1000);
    await settled;
  });

  it("should not surface a late rejection of the request", async () => {
    const rejections: unknown[] = [];
    const onRejection = (reason: unknown) => rejections.push(reason);
    process.on("unhandledRejection", onRejection);
    try {
      const request = Promise.withResolvers<number>();
      const late = withDeadline(request.promise, TimeSpan.seconds(1), () => "too slow");
      const settled = expect(late).rejects.toThrow("too slow");
      await vi.advanceTimersByTimeAsync(1000);
      await settled;
      request.reject(new Error("fetch exploded"));
      vi.useRealTimers();
      await new Promise((resolve) => setTimeout(resolve, 10));
    } finally {
      process.off("unhandledRejection", onRejection);
    }
    expect(rejections).toEqual([]);
  });
});
