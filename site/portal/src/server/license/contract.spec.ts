// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createPublicKey, generateKeyPairSync, verify } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

import { beforeAll, describe, expect, it } from "vitest";

import { build } from "@/server/license/claims";
import { local, sign } from "@/server/license/sign";
import { HASH_A, LICENSE, NOW } from "@/server/license/testutil";

// The Core's license spec reads the same fixture. Regenerate both files with
// CONTRACT_REGENERATE=1 pnpm exec vitest run src/server/license/contract.spec.ts
const TESTDATA = path.resolve(
  import.meta.dirname,
  "../../../../../core/pkg/service/license/testdata",
);
const PUBLIC_KEY = path.join(TESTDATA, "portal.pub");
const KEY = path.join(TESTDATA, "portal.license");
const KID = "contract";
const RAW_PUBLIC_KEY_LENGTH = 1312;

const INPUTS = {
  license: { ...LICENSE, channels: 50, maxVersion: "0.62" },
  fingerprint: [HASH_A],
  now: NOW,
};

// Every ML-DSA-44 SPKI shares one prefix before the raw public key.
const SPKI_PREFIX = generateKeyPairSync("ml-dsa-44")
  .publicKey.export({ format: "der", type: "spki" })
  .subarray(0, -RAW_PUBLIC_KEY_LENGTH);

const regenerate = async (): Promise<void> => {
  const { privateKey, publicKey } = generateKeyPairSync("ml-dsa-44");
  const spki = publicKey.export({ format: "der", type: "spki" });
  writeFileSync(
    PUBLIC_KEY,
    `${spki.subarray(-RAW_PUBLIC_KEY_LENGTH).toString("base64")}\n`,
  );
  writeFileSync(KEY, `${await sign(local(privateKey, KID), build(INPUTS))}\n`);
};

const decode = (segment: string): unknown =>
  JSON.parse(Buffer.from(segment, "base64url").toString());

describe("license key contract", () => {
  let parts: string[];
  let publicKey: ReturnType<typeof createPublicKey>;

  beforeAll(async () => {
    if (process.env.CONTRACT_REGENERATE === "1") await regenerate();
    parts = readFileSync(KEY, "utf8").trim().split(".");
    const raw = Buffer.from(readFileSync(PUBLIC_KEY, "utf8").trim(), "base64");
    publicKey = createPublicKey({
      key: Buffer.concat([SPKI_PREFIX, raw]),
      format: "der",
      type: "spki",
    });
  });

  it("should hold a signature the fixture public key verifies", () => {
    const [header, payload, signature] = parts;
    expect(
      verify(
        null,
        Buffer.from(`${header}.${payload}`),
        publicKey,
        Buffer.from(signature, "base64url"),
      ),
    ).toBe(true);
  });

  it("should carry the payload the portal signs today", async () => {
    const [, payload] = (
      await sign({ kid: KID, sign: async () => new Uint8Array() }, build(INPUTS))
    ).split(".");
    expect(decode(parts[1])).toEqual(decode(payload));
  });

  it("should name the contract key in its header", () => {
    expect(decode(parts[0])).toEqual({ alg: "ML-DSA-44", typ: "JWT", kid: KID });
  });
});
