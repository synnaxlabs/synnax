// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan } from "@synnaxlabs/x";
import { assertDefined } from "@synnaxlabs/x/testutil";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { AuthError, type Synnax } from "@/index";
import {
  createRequestHold,
  createTestClient,
  holdRequests,
  rejectRequests,
} from "@/testutil";

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

describe("createTestClient lifetime", () => {
  it("should close a client made inside a test once the test finishes", async () => {
    const made: { client?: Synnax } = {};
    // Finished hooks run in stack order, so this one runs after the client closes.
    onTestFinished(async () => {
      assertDefined(made.client);
      // Closing the client ends its session, so it can no longer sign its requests.
      const err = await made.client.channels
        .retrieve(["no-such-channel"])
        .catch((e: unknown) => e);
      expect(AuthError.matches(err)).toBe(true);
    });
    made.client = createTestClient();
    await made.client.connect();
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
    await sleep.sleep(TimeSpan.milliseconds(50));
    expect(reachedCore()).toBe(false);
    release();
    await expect(retrieving).resolves.toEqual([]);
    expect(reachedCore()).toBe(true);
  });
});

describe("createRequestHold", () => {
  it("should hold a matching request from engage to release and send the others", async () => {
    const hold = createRequestHold((target) => target.endsWith("/channel/retrieve"));
    const client = createTestClient({ middleware: [hold.middleware] });
    await expect(client.channels.retrieve(["no-such-channel"])).resolves.toEqual([]);
    hold.engage();
    let settled = false;
    const retrieving = client.channels
      .retrieve(["no-such-channel"])
      .finally(() => (settled = true));
    await expect(client.ranges.retrieve({ searchTerm: "rng" })).resolves.toBeDefined();
    expect(settled).toBe(false);
    hold.release();
    await expect(retrieving).resolves.toEqual([]);
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
