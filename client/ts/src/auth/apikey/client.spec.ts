// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { apiKey } from "@/auth/apikey";
import { InvalidCredentialsError, InvalidTokenError, ValidationError } from "@/index";
import { createTestAuthClient, createTestClient } from "@/testutil";

describe("apiKey", () => {
  it("should create a key, log in with it, list it, and delete it", async () => {
    const owner = createTestClient().auth;
    const { token, key } = await owner.apiKeys.create({ name: "spec" });
    expect(token.startsWith("syk_")).toBe(true);
    expect(key.name).toEqual("spec");
    expect(key.subject).toEqual(owner.subject);
    const keyed = createTestAuthClient({ apiKey: token });
    await keyed.sessions.retrieve();
    expect(keyed.session?.credential).toEqual(key.key);
    expect(keyed.subject).toEqual(owner.subject);
    expect((await owner.apiKeys.retrieve()).map((k) => k.key)).toContain(key.key);
    await owner.apiKeys.delete(key.key);
    expect((await owner.apiKeys.retrieve()).map((k) => k.key)).not.toContain(key.key);
  });

  it("should read the record key from a token", async () => {
    const owner = createTestClient().auth;
    const { token, key } = await owner.apiKeys.create({ name: "id" });
    expect(apiKey.keyOf(token)).toEqual(key.key);
    await owner.apiKeys.delete(key.key);
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

  it("should fail the next request of a client whose key was deleted", async () => {
    const owner = createTestClient().auth;
    const { token, key } = await owner.apiKeys.create({ name: "deleted" });
    const keyed = createTestAuthClient({ apiKey: token });
    await keyed.sessions.retrieve();
    await owner.apiKeys.delete(key.key);
    await expect(keyed.sessions.retrieve()).rejects.toThrow(InvalidCredentialsError);
  });

  describe("expiry", () => {
    it("should create a key that never expires by default", async () => {
      const owner = createTestClient().auth;
      const { key } = await owner.apiKeys.create({ name: "never" });
      expect(key.expires).toBeUndefined();
      await owner.apiKeys.delete(key.key);
    });

    it("should create a key with an expiry", async () => {
      const owner = createTestClient().auth;
      const expires = TimeStamp.now().add(TimeSpan.days(90));
      const { key } = await owner.apiKeys.create({ name: "expiring", expires });
      expect(key.expires?.equals(expires)).toBe(true);
      const [retrieved] = (await owner.apiKeys.retrieve()).filter(
        (k) => k.key === key.key,
      );
      expect(retrieved.expires?.equals(expires)).toBe(true);
      await owner.apiKeys.delete(key.key);
    });

    it("should refuse an expiry in the past", async () => {
      const owner = createTestClient().auth;
      await expect(
        owner.apiKeys.create({
          name: "past",
          expires: TimeStamp.now().sub(TimeSpan.SECOND),
        }),
      ).rejects.toThrow(ValidationError);
    });

    it("should fail the next request of a client whose key expired", async () => {
      const owner = createTestClient().auth;
      const { token, key } = await owner.apiKeys.create({
        name: "short",
        expires: TimeStamp.now().add(TimeSpan.seconds(1)),
      });
      const keyed = createTestAuthClient({ apiKey: token });
      await keyed.sessions.retrieve();
      await sleep.sleep(TimeSpan.milliseconds(1500));
      await expect(keyed.sessions.retrieve()).rejects.toThrow(InvalidCredentialsError);
      await owner.apiKeys.delete(key.key);
    });
  });
});
