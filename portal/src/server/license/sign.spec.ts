// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { generateKeyPairSync, verify } from "node:crypto";

import { license } from "@synnaxlabs/client";
import { binary } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { local, sign } from "@/server/license/sign";
import { CLAIMS } from "@/server/license/testutil";

const decode = (segment: string): unknown =>
  JSON.parse(Buffer.from(segment, "base64url").toString());

describe("sign", () => {
  const { privateKey, publicKey } = generateKeyPairSync("ed25519");
  const signer = local(privateKey, "test");

  it("should produce a compact JWS the public key verifies", async () => {
    const token = await sign(signer, CLAIMS);
    const [header, payload, signature] = token.split(".");
    const input = Buffer.from(`${header}.${payload}`);
    expect(verify(null, input, publicKey, Buffer.from(signature, "base64url"))).toBe(
      true,
    );
  });

  it("should name the algorithm and key in the header", async () => {
    const [header] = (await sign(signer, CLAIMS)).split(".");
    expect(decode(header)).toEqual({ alg: "EdDSA", typ: "JWT", kid: "test" });
  });

  it("should key the claims as the Core reads them", async () => {
    const [, payload] = (await sign(signer, { ...CLAIMS, maxVersion: "0.62" })).split(
      ".",
    );
    expect(Object.keys(decode(payload) as object).sort()).toEqual([
      "channels",
      "claims_version",
      "edition",
      "exp",
      "fingerprint_scheme",
      "fingerprints",
      "iat",
      "jti",
      "machines",
      "max_version",
      "organization",
      "required",
    ]);
  });

  it("should carry claims the client codec decodes back", async () => {
    const [, payload] = (await sign(signer, CLAIMS)).split(".");
    const json = Buffer.from(payload, "base64url").toString();
    expect(binary.JSON_CODEC.decodeString(json, license.licenseZ)).toEqual(CLAIMS);
  });

  it("should omit absent optional claims instead of writing null", async () => {
    const [, payload] = (await sign(signer, { ...CLAIMS, exp: undefined })).split(".");
    expect(decode(payload)).not.toHaveProperty("exp");
  });
});
