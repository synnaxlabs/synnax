// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AuthError } from "@/errors";
import { createTestClient } from "@/testutil";

const createKeyedClient = (apiKey: string) =>
  createTestClient({ username: undefined, password: undefined, apiKey });

describe("apiKey", () => {
  it("should create a key, log in with it, list it, and delete it", async () => {
    const owner = createTestClient();
    const { token, key } = await owner.auth.apiKeys.create({ name: "spec" });
    expect(token.startsWith("syk_")).toBe(true);
    expect(key.name).toEqual("spec");
    const keyed = createKeyedClient(token);
    await keyed.auth.sessions.retrieve();
    expect(keyed.auth.session?.credential).toEqual(key.key);
    expect(keyed.auth.subject).toEqual(owner.auth.subject);
    expect(key.subject).toEqual(owner.auth.subject);
    const before = await owner.auth.apiKeys.retrieve();
    expect(before.map((k) => k.key)).toContain(key.key);
    await owner.auth.apiKeys.delete(key.key);
    const after = await owner.auth.apiKeys.retrieve();
    expect(after.map((k) => k.key)).not.toContain(key.key);
  });

  it("should fail the next request of a client whose key was deleted", async () => {
    const owner = createTestClient();
    const { token, key } = await owner.auth.apiKeys.create({ name: "deleted" });
    const keyed = createKeyedClient(token);
    await keyed.auth.sessions.retrieve();
    await owner.auth.apiKeys.delete(key.key);
    await expect(keyed.auth.sessions.retrieve()).rejects.toThrow(AuthError);
  });
});
