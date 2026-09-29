// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { createFetcher } from "@/util/checks/fetch";

const LINK = "https://example.com/page";

const response = (status: number): Response =>
  ({ status, body: null, text: async () => "" }) as unknown as Response;

// Each attempt is a HEAD, then a GET when the HEAD fails.
const stub = (...statuses: number[]): ReturnType<typeof vi.fn> => {
  const fetch = vi.fn();
  statuses.forEach((status) => fetch.mockResolvedValueOnce(response(status)));
  fetch.mockResolvedValue(response(statuses[statuses.length - 1]));
  vi.stubGlobal("fetch", fetch);
  return fetch;
};

describe("createFetcher", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("should pass a link once a 503 clears", async () => {
    stub(503, 503, 200);
    const result = createFetcher("http://localhost:4399")(LINK);
    await vi.advanceTimersByTimeAsync(15_000);
    expect(await result).toBeNull();
  });

  it("should wait longer after a 503 than after another failure", async () => {
    const unavailable = stub(503);
    void createFetcher("http://localhost:4399")(LINK);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(unavailable).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(14_000);
    expect(unavailable).toHaveBeenCalledTimes(4);
  });

  it("should retry another failure after a second", async () => {
    const broken = stub(500);
    void createFetcher("http://localhost:4399")(LINK);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(broken).toHaveBeenCalledTimes(4);
  });

  it("should fail a link that stays 503", async () => {
    stub(503);
    const result = createFetcher("http://localhost:4399")(LINK);
    await vi.advanceTimersByTimeAsync(45_000);
    expect(await result).toContain("HTTP 503");
  });
});
