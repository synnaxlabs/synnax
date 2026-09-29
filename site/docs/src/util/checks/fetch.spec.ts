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

const networkError = (code: string): Error =>
  new TypeError("fetch failed", { cause: Object.assign(new Error(code), { code }) });

describe("createFetcher", () => {
  let warnings: string[];
  const fetcher = (): ((url: string) => Promise<string | null>) =>
    createFetcher("http://localhost:4399", (message) => warnings.push(message));

  beforeEach(() => {
    vi.useFakeTimers();
    warnings = [];
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("should pass a link once a 503 clears", async () => {
    stub(503, 503, 200);
    const result = fetcher()(LINK);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(await result).toBeNull();
    expect(warnings).toEqual([]);
  });

  it("should warn instead of failing on a link that stays 503", async () => {
    stub(503);
    const result = fetcher()(LINK);
    await vi.advanceTimersByTimeAsync(3_000);
    expect(await result).toBeNull();
    expect(warnings).toEqual([`${LINK}: HTTP 503`]);
  });

  it("should warn instead of failing on a dropped connection", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(networkError("ECONNRESET")));
    const result = fetcher()(LINK);
    await vi.advanceTimersByTimeAsync(3_000);
    expect(await result).toBeNull();
    expect(warnings).toEqual([`${LINK}: ECONNRESET`]);
  });

  it("should fail a link that answers 404", async () => {
    stub(404);
    expect(await fetcher()(LINK)).toBe(`${LINK}: HTTP 404`);
    expect(warnings).toEqual([]);
  });

  it("should fail a link whose host does not resolve", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(networkError("ENOTFOUND")));
    const result = fetcher()(LINK);
    await vi.advanceTimersByTimeAsync(3_000);
    expect(await result).toBe(`${LINK}: ENOTFOUND`);
  });

  it("should warn instead of failing on a host its breaker skipped", async () => {
    const hung = vi.fn().mockRejectedValue(networkError("ECONNRESET"));
    vi.stubGlobal("fetch", hung);
    const fetchOk = fetcher();
    for (const path of ["a", "b", "c"]) {
      const result = fetchOk(`${LINK}/${path}`);
      await vi.advanceTimersByTimeAsync(3_000);
      await result;
    }
    const calls = hung.mock.calls.length;
    expect(await fetchOk(`${LINK}/d`)).toBeNull();
    expect(hung).toHaveBeenCalledTimes(calls);
    expect(warnings).toContain(`${LINK}/d: skipped, example.com stopped answering`);
  });

  it("should fail a link to a missing anchor", async () => {
    stub(200);
    expect(await fetcher()(`${LINK}#absent`)).toBe(
      `${LINK}#absent: missing anchor #absent`,
    );
  });
});
