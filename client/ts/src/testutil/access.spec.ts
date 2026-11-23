// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware } from "@synnaxlabs/freighter";
import { describe, expect, it, vi } from "vitest";

import {
  createRequestHold,
  createTestClient,
  createTestClientWithPolicy,
} from "@/testutil";
import { user } from "@/user";

const POLICY_RETRIEVE = "/access/policy/retrieve";

describe("createTestClientWithPolicy", () => {
  it("should build the client with the given middleware", async () => {
    const middleware = vi.fn<Middleware>(async (ctx, next) => await next(ctx));
    await createTestClientWithPolicy(
      createTestClient(),
      { name: "test", objects: [user.TYPE_RESOURCE_ID], actions: ["retrieve"] },
      undefined,
      { middleware: [middleware] },
    );
    const targets = middleware.mock.calls.map(([ctx]) => ctx.target);
    expect(targets.some((t) => t.endsWith(POLICY_RETRIEVE))).toBe(true);
  });

  it("should finish its setup reads through a hold that is not engaged", async () => {
    const hold = createRequestHold((target) => target.endsWith(POLICY_RETRIEVE));
    const restricted = await createTestClientWithPolicy(
      createTestClient(),
      { name: "test", objects: [user.TYPE_RESOURCE_ID], actions: ["retrieve"] },
      undefined,
      { middleware: [hold.middleware] },
    );
    hold.engage();
    let settled = false;
    const me = await restricted.users.current.retrieve();
    const retrieving = restricted.access.policies
      .retrieve({ subjects: [user.resourceID(me.key)] })
      .finally(() => (settled = true));
    await expect(restricted.users.current.retrieve()).resolves.toBeDefined();
    expect(settled).toBe(false);
    hold.release();
    await expect(retrieving).resolves.toBeDefined();
  });
});
