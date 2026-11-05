// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, onTestFinished, vi } from "vitest";

import { createTestClient } from "@/testutil/client";

describe("createTestClient", () => {
  it("should retry a request without writing to the console", async () => {
    const client = createTestClient();
    const realFetch = globalThis.fetch;
    let failures = 0;
    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation(async (input, init) => {
        if (failures === 0 && String(input).includes("channel/retrieve")) {
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
