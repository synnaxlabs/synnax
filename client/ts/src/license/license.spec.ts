// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AccessDeniedError, InvalidLicenseError } from "@/errors";
import { license } from "@/license";
import { ontology } from "@/ontology";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("license", () => {
  it("should retrieve the Core's license state", async () => {
    const info = await client.license.retrieve();
    for (const hash of info.fingerprint) expect(hash).toMatch(/^[0-9a-f]{64}$/);
    expect(info.license == null).toBe(info.state === "missing");
  });

  it("should reject a license key that cannot be verified", async () => {
    await expect(client.license.activate("not-a-license-key")).rejects.toThrow(
      InvalidLicenseError,
    );
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
