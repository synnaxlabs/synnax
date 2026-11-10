// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, onTestFinished, vi } from "vitest";

import { createTestClient, holdRequests, rejectRequests } from "@/testutil/client";

describe("createTestClient", () => {
  it("should retry a request without writing to the console", async () => {
    const client = createTestClient();
    const realFetch = globalThis.fetch;
    let failures = 0;
    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation(async (input, init) => {
        if (
          failures === 0 &&
          typeof input === "string" &&
          input.includes("channel/retrieve")
        ) {
          failures++;
          throw new TypeError("fetch failed", {
            cause: { code: "UND_ERR_SOCKET" },
          });
        }
        return await realFetch(input, init);
      });
    onTestFinished(() => fetchSpy.mockRestore());
    const warn = vi.spyOn(console, "warn");
    onTestFinished(() => warn.mockRestore());
    await expect(client.channels.retrieve(["no-such-channel"])).resolves.toEqual([]);
    expect(failures).toBe(1);
    expect(warn).not.toHaveBeenCalled();
  });
});

describe("holdRequests", () => {
  it("should send a held request only once released", async () => {
    const client = createTestClient();
    await client.connect();
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    onTestFinished(() => fetchSpy.mockRestore());
    const { release } = holdRequests(client, "/channel/retrieve");
    const retrieving = client.channels.retrieve(["no-such-channel"]);
    const reachedCore = () =>
      fetchSpy.mock.calls.some(
        ([input]) => typeof input === "string" && input.includes("channel/retrieve"),
      );
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(reachedCore()).toBe(false);
    release();
    await expect(retrieving).resolves.toEqual([]);
    expect(reachedCore()).toBe(true);
  });
});

describe("rejectRequests", () => {
  it("should reject only the requests that match", async () => {
    const client = createTestClient();
    const spy = rejectRequests(
      client,
      (target) => target === "/range/retrieve",
      new Error("boom"),
    );
    onTestFinished(() => spy.mockRestore());
    await expect(client.ranges.retrieve({ searchTerm: "rng" })).rejects.toThrow("boom");
    await expect(client.channels.retrieve(["no-such-channel"])).resolves.toEqual([]);
  });
});
