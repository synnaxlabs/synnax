// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyObject, sign as nodeSign } from "node:crypto";

import { type KMSClient, SignCommand } from "@aws-sdk/client-kms";
import { type license as client } from "@synnaxlabs/client";

/** Signer produces a raw Ed25519 signature over a JWS signing input. */
export interface Signer {
  /** kid names the public key a Core verifies the signature with. */
  kid: string;
  sign: (input: Uint8Array) => Promise<Uint8Array>;
}

const base64url = (data: Uint8Array | string): string =>
  Buffer.from(data).toString("base64url");

/** sign encodes the claims as a compact EdDSA JWS. */
export const sign = async (signer: Signer, claims: client.License): Promise<string> => {
  const header = base64url(
    JSON.stringify({ alg: "EdDSA", typ: "JWT", kid: signer.kid }),
  );
  const payload = base64url(JSON.stringify(claims));
  const input = `${header}.${payload}`;
  const signature = await signer.sign(new TextEncoder().encode(input));
  return `${input}.${base64url(signature)}`;
};

export interface KMSArgs {
  client: KMSClient;
  keyID: string;
  kid: string;
}

/** kms signs with an Ed25519 key that never leaves AWS KMS. */
export const kms = ({ client, keyID, kid }: KMSArgs): Signer => ({
  kid,
  sign: async (input) => {
    const out = await client.send(
      new SignCommand({
        KeyId: keyID,
        Message: input,
        MessageType: "RAW",
        SigningAlgorithm: "ED25519_SHA_512",
      }),
    );
    if (out.Signature == null) throw new Error("KMS returned no signature");
    return out.Signature;
  },
});

/** local signs with an in-process Ed25519 private key. For tests only. */
export const local = (privateKey: KeyObject, kid: string): Signer => ({
  kid,
  sign: async (input) => new Uint8Array(nodeSign(null, input, privateKey)),
});
