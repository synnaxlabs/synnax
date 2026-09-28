// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { InvalidLicenseError } from "@/errors";
import { license } from "@/license";
import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("license", () => {
  it("should retrieve the Core's license state", async () => {
    const info = await client.license.retrieve();
    expect(license.STATES).toContain(info.state);
    expect(Array.isArray(info.fingerprint)).toBe(true);
  });

  it("should reject a token that cannot be verified", async () => {
    await expect(client.license.activate("not-a-token")).rejects.toThrow(
      InvalidLicenseError,
    );
  });
});
