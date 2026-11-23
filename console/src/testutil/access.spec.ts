// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Middleware } from "@synnaxlabs/freighter";
import { describe, expect, it, vi } from "vitest";

import { createTestClientWithGrants } from "@/testutil";

describe("createTestClientWithGrants", () => {
  it("should build the client with the given parameters", async () => {
    const middleware = vi.fn<Middleware>(async (ctx, next) => await next(ctx));
    const granted = await createTestClientWithGrants(
      createTestClient(),
      { retrieve: [ranger.TYPE_RESOURCE_ID] },
      { middleware: [middleware] },
    );
    await granted.ranges.retrieve({ searchTerm: "rng" });
    const targets = middleware.mock.calls.map(([ctx]) => ctx.target);
    expect(targets.some((t) => t.endsWith("/range/retrieve"))).toBe(true);
  });
});
