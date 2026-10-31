// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { type Clerk, logOut } from "@/ui/clerk";

describe("logOut", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("should make PostHog forget the user before it moves on", async () => {
    const calls: string[] = [];
    vi.stubGlobal("window", {
      posthog: { __loaded: true, reset: () => calls.push("reset") },
    });
    const clerk = {
      signOut: async (after: () => Promise<void>) => {
        calls.push("sign out");
        await after();
      },
    } as unknown as Clerk;
    logOut(clerk, () => calls.push("then"));
    await vi.waitFor(() => expect(calls).toEqual(["sign out", "reset", "then"]));
  });
});
