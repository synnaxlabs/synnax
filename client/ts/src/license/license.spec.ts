// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { connection } from "@/connection";
import { AccessDeniedError, InvalidLicenseError } from "@/errors";
import { license } from "@/license";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

const ACTIVATED: license.Info = { state: "ok", warning: "", fingerprint: ["a"] };

/** A license client whose Core answers every request with info. */
const createClient = (info: license.Info): license.Client =>
  new license.Client({
    unary: { send: vi.fn().mockResolvedValue(info), use: vi.fn() },
    connection: {
      status: connection.DEFAULT_STATUS,
      onChange: () => () => {},
      retryNow: vi.fn(),
    },
    cache: new query.Cache({ openStreamer: null }),
  });

describe("license", () => {
  it("should retrieve the Core's license state", async () => {
    const info = await client.license.retrieve();
    for (const hash of info.fingerprint) expect(hash).toMatch(/^[0-9a-f]{64}$/);
    expect(info.license == null).toBe(info.state === "missing");
  });

  it("should keep the license when deactivating a key the Core does not hold", async () => {
    const before = await client.license.retrieve();
    const after = await client.license.deactivate(
      "8f0f3c52-4c6a-4d6c-9a33-2e4b6a1f7d10",
    );
    expect(after.state).toBe(before.state);
  });

  it("should reject a license key that cannot be verified", async () => {
    await expect(client.license.activate("not-a-license-key")).rejects.toThrow(
      InvalidLicenseError,
    );
  });

  describe("cache", () => {
    it("should cache nothing before the first read", () => {
      expect(createClient(ACTIVATED).getCached()).toBeUndefined();
    });

    it("should cache the license a read returns", async () => {
      const licenses = createClient(ACTIVATED);
      const info = await licenses.retrieve();
      expect(info).toEqual(ACTIVATED);
      expect(licenses.getCached()).toEqual(ACTIVATED);
    });

    it("should cache an activated license", async () => {
      const licenses = createClient(ACTIVATED);
      await licenses.activate("key");
      expect(licenses.getCached()).toEqual(ACTIVATED);
    });

    it("should cache the state a deactivation returns", async () => {
      const missing: license.Info = {
        state: "missing",
        warning: "",
        fingerprint: ["a"],
      };
      const licenses = createClient(missing);
      await licenses.deactivate("8f0f3c52-4c6a-4d6c-9a33-2e4b6a1f7d10");
      expect(licenses.getCached()).toEqual(missing);
    });

    it("should notify subscribers of an activated license", async () => {
      const licenses = createClient(ACTIVATED);
      const handler = vi.fn();
      const disconnect = licenses.onChange(handler);
      await licenses.activate("key");
      expect(handler).toHaveBeenCalledWith(ACTIVATED);
      disconnect();
    });
  });

  describe("access", () => {
    it("should read the license with a retrieve grant on its ID", async () => {
      const granted = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [license.ONTOLOGY_ID],
        actions: ["retrieve"],
      });
      await expect(granted.license.retrieve()).resolves.toBeDefined();
    });

    it("should deny the license without a grant on its ID", async () => {
      const denied = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [ontology.ROOT_ID],
        actions: ["retrieve"],
      });
      await expect(denied.license.retrieve()).rejects.toThrow(AccessDeniedError);
    });
  });
});
