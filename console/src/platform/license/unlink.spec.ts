// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { License } from "@/platform/license";

describe("unlink", () => {
  afterEach(() => vi.unstubAllGlobals());

  const answer = (status: number, statusText = ""): void => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(null, { status, statusText })),
    );
  };

  it("should post the secret as a bearer token, with a deadline", async () => {
    answer(204);
    await License.unlink("shh");
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(License.UNLINK_URL, {
      method: "POST",
      headers: { authorization: "Bearer shh" },
      signal: expect.any(AbortSignal),
    });
  });

  it("should throw when the hub refuses", async () => {
    answer(403, "Forbidden");
    await expect(License.unlink("shh")).rejects.toThrow(
      "Could not log out: 403 Forbidden",
    );
  });
});
