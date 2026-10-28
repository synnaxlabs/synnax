// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { createTestClient, createTestUser } from "@/testutil";
import { user } from "@/user";

describe("session", () => {
  describe("retrieve", () => {
    it("should list the caller's own session by default", async () => {
      const client = createTestClient();
      const sessions = await client.auth.sessions.retrieve();
      expect(sessions.map((s) => s.key)).toContain(client.auth.session?.key);
    });

    it("should list the sessions of another subject", async () => {
      const owner = createTestClient();
      const { user: u, username, password } = await createTestUser(owner);
      const other = createTestClient({ username, password });
      await other.auth.sessions.retrieve();
      const sessions = await owner.auth.sessions.retrieve([user.ontologyID(u.key)]);
      expect(sessions.map((s) => s.key)).toContain(other.auth.session?.key);
    });
  });

  describe("delete", () => {
    it("should end a session, so the next request logs in again", async () => {
      const client = createTestClient();
      await client.auth.sessions.retrieve();
      const key = client.auth.session?.key;
      if (key == null) throw new Error("no session");
      await client.auth.sessions.delete(key);
      await client.auth.sessions.retrieve();
      expect(client.auth.session?.key).not.toEqual(key);
    });
  });
});
