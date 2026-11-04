// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { apiKey } from "@/auth/apikey";
import { AuthError, InvalidTokenError } from "@/errors";
import { createTestAuthClient, createTestClient } from "@/testutil";

describe("apiKey", () => {
  it("should create a key, log in with it, list it, and delete it", async () => {
    const owner = createTestClient();
    const { token, key } = await owner.auth.apiKeys.create({ name: "spec" });
    expect(token.startsWith("syk_")).toBe(true);
    expect(key.name).toEqual("spec");
    const keyed = createTestAuthClient({ apiKey: token });
    await keyed.sessions.retrieve();
    expect(keyed.session?.credential).toEqual(key.key);
    expect(keyed.subject).toEqual(owner.auth.subject);
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
    const keyed = createTestAuthClient({ apiKey: token });
    await keyed.sessions.retrieve();
    await owner.auth.apiKeys.delete(key.key);
    await expect(keyed.sessions.retrieve()).rejects.toThrow(AuthError);
  });

  it("should read the record key from a token", async () => {
    const client = createTestClient();
    const { token, key } = await client.auth.apiKeys.create({ name: "id" });
    expect(apiKey.keyOf(token)).toEqual(key.key);
    await client.auth.apiKeys.delete(key.key);
  });

  it("should refuse a token that is not an API key", () => {
    expect(() => apiKey.keyOf("sys_abc")).toThrow(
      new InvalidTokenError("not an API key"),
    );
  });

  it("should refuse an API key of the wrong length", () => {
    expect(() => apiKey.keyOf("syk_c2hvcnQ")).toThrow(
      new InvalidTokenError("malformed API key"),
    );
  });

  it("should refuse an API key that is not base64", () => {
    expect(() => apiKey.keyOf("syk_!!!")).toThrow(
      new InvalidTokenError("malformed API key"),
    );
  });
});
