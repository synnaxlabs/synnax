// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { query } from "@/query";
import {
  createTestClient,
  createTestClientWithRole,
  TEST_CREDENTIALS,
} from "@/testutil";

const client = createTestClient();

const { username, password } = TEST_CREDENTIALS;

const signedInKey = async () => (await client.users.retrieve({ username })).key;

describe("current", () => {
  it("should retrieve the user before the client connects", async () => {
    const fresh = createTestClient();
    const u = await fresh.users.current.retrieve();
    expect(u.key).toEqual(await signedInKey());
  });

  it("should deliver the user to a subscriber that subscribed before the login", async () => {
    const fresh = createTestClient();
    const handler = vi.fn();
    const off = fresh.users.current.onChange(handler);
    try {
      expect(fresh.users.current.getCached()).toBeUndefined();
      await fresh.connect();
      expect(handler).toHaveBeenLastCalledWith(
        expect.objectContaining({ key: await signedInKey() }),
      );
    } finally {
      off();
    }
  });

  it("should deliver a rename from another client", async () => {
    const viewer = await createTestClientWithRole(client, "Viewer");
    // Changes made while the stream is still opening are lost, so open it first.
    await viewer.connect();
    const handler = vi.fn();
    const off = viewer.users.current.onChange(handler);
    try {
      const { key } = await viewer.users.current.retrieve();
      await client.users.rename(key, "Abigail Adams");
      await expect
        .poll(() => {
          const cached = viewer.users.current.getCached();
          return query.isLive(cached) && cached.name === "Abigail Adams";
        })
        .toBe(true);
      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({ name: "Abigail Adams" }),
      );
    } finally {
      off();
    }
  });

  it("should follow a login as a different user", async () => {
    const viewer = await createTestClientWithRole(client, "Viewer");
    const handler = vi.fn();
    const off = viewer.users.current.onChange(handler);
    try {
      viewer.reauthenticate({ username, password });
      expect(viewer.users.current.getCached()).toBeUndefined();
      const u = await viewer.users.current.retrieve();
      expect(u.key).toEqual(await signedInKey());
      expect(handler).toHaveBeenLastCalledWith(u);
    } finally {
      off();
    }
  });
});
