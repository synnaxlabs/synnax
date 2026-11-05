// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp } from "@synnaxlabs/x";
import { assert, describe, expect, it } from "vitest";

import { NotFoundError, PathError, ValidationError } from "@/index";
import { rack } from "@/rack";
import { createTestClient, waitForStreamLive } from "@/testutil";

// A self-signed P-256 certificate valid until 2126, its key, and an unrelated key.
const PEM = `-----BEGIN CERTIFICATE-----
MIIBnDCCAUKgAwIBAgIUTlojETDboAh4nmfahlzAemoZ4g4wCgYIKoZIzj0EAwIw
MzEbMBkGA1UEAwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXgg
TGFiczAgFw0yNjExMDQwMjUyMTBaGA8yMTI2MTAxMTAyNTIxMFowMzEbMBkGA1UE
AwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXggTGFiczBZMBMG
ByqGSM49AgEGCCqGSM49AwEHA0IABAttV7/ZZaP/ck5ZSEibyz8Bq/FOha2BqRFx
KcHLLjh+GByF19Xf9tZdf+9TfDlOPwyGPp+zArW6WOjeHdzzPv6jMjAwMB0GA1Ud
DgQWBBQnJ8GnAtkFwtBOUq0b4+pWyuXWQTAPBgNVHRMBAf8EBTADAQH/MAoGCCqG
SM49BAMCA0gAMEUCIQDucqkUixw8A44mdBmPFZIhYtq3tzNMzPF5zPD9ONNZPwIg
I3WFxQbCO86E+bgOOHH1NSjmhCFtORY5jSOGw6yL3m4=
-----END CERTIFICATE-----
`;
const PRIVATE_KEY = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgjJtOV99NCzRnmMoB
wui9Lv+9a8aqZ+LwPX2PyGfVEa+hRANCAAQLbVe/2WWj/3JOWUhIm8s/AavxToWt
gakRcSnByy44fhgchdfV3/bWXX/vU3w5Tj8Mhj6fswK1uljo3h3c8z7+
-----END PRIVATE KEY-----
`;
const OTHER_PRIVATE_KEY = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgCEpmg8mYhXe2mUBp
Zj5oeHcm37uA23Gal/ivbSzbF2ShRANCAAQoi1G7u+dFDR3/oMHvF2swFlLtpA9e
SUHP7pVz5CuFmHrTXcGKjGaN0e54b+eQR3sHOokr9B46aGQon8U6WZ1E
-----END PRIVATE KEY-----
`;
const SHA1 = "c67f3f9384d557875ab3dc59bea3d9e11239ea73";
const SHA256 = "20637df2df608327ed9cd031811c822513256444225a7bf655559f0baee4a970";

describe("Certificate", () => {
  describe("create", () => {
    it("should read the details of a stored certificate", async () => {
      const client = createTestClient();
      const c = await client.certificates.create({
        name: "Broker CA",
        source: "stored",
        pem: PEM,
      });
      expect(c).toMatchObject({ name: "Broker CA", source: "stored", pem: PEM });
      assert(c.source === "stored");
      expect(c.privateKey).toBeUndefined();
      expect(c.details).toMatchObject({ sha1: SHA1, sha256: SHA256 });
      expect(c.details.subject).toBe("CN=Synnax Test Client,O=Synnax Labs");
      expect(c.details.issuer).toBe(c.details.subject);
      expect(c.details.notAfter.after(TimeStamp.now())).toBe(true);
      expect(c.details.notBefore.before(c.details.notAfter)).toBe(true);
    });

    it("should save a private key as a secret", async () => {
      const client = createTestClient();
      const c = await client.certificates.create({
        name: "Client certificate",
        source: "stored",
        pem: PEM,
        privateKey: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const key = await client.secrets.retrieve(c.privateKey);
      expect(key.source).toBe("stored");
    });

    it("should reject a private key that does not match the certificate", async () => {
      const client = createTestClient();
      const create = client.certificates.create({
        name: "Mismatched",
        source: "stored",
        pem: PEM,
        privateKey: OTHER_PRIVATE_KEY,
      });
      await expect(create).rejects.toThrow(PathError);
      await expect(create).rejects.toThrow(
        "private_key: does not match the certificate: validation error",
      );
    });

    it("should create a file certificate", async () => {
      const client = createTestClient();
      const c = await client.certificates.create({
        name: "On disk",
        source: "file",
        certificatePath: "/etc/synnax/client.crt",
        privateKeyPath: "/etc/synnax/client.key",
      });
      expect(c).toMatchObject({
        name: "On disk",
        source: "file",
        certificatePath: "/etc/synnax/client.crt",
        privateKeyPath: "/etc/synnax/client.key",
      });
    });
  });

  describe("retrieve", () => {
    it("should retrieve certificates by key", async () => {
      const client = createTestClient();
      const created = await client.certificates.create([
        { name: "Retrieve 1", source: "stored", pem: PEM },
        { name: "Retrieve 2", source: "file", certificatePath: "/tmp/a.crt" },
      ]);
      const keys = created.map((c) => c.key);
      const retrieved = await createTestClient().certificates.retrieve({ keys });
      expect(retrieved).toHaveLength(2);
      expect(await createTestClient().certificates.retrieve(keys[0])).toEqual(
        created[0],
      );
    });

    it("should search for certificates by name", async () => {
      const client = createTestClient();
      const name = `${id.create()} CA`;
      const created = await client.certificates.create({
        name,
        source: "stored",
        pem: PEM,
      });
      await expect
        .poll(async () => await client.certificates.retrieve({ searchTerm: name }))
        .toContainEqual(created);
    });
  });

  describe("rename", () => {
    it("should rename a certificate", async () => {
      const client = createTestClient();
      const c = await client.certificates.create({
        name: "Before",
        source: "stored",
        pem: PEM,
      });
      await client.certificates.rename(c.key, "After");
      const retrieved = await createTestClient().certificates.retrieve(c.key);
      expect(retrieved).toMatchObject({ name: "After", pem: PEM });
    });
  });

  describe("delete", () => {
    it("should delete a certificate", async () => {
      const client = createTestClient();
      const c = await client.certificates.create({
        name: "Delete",
        source: "stored",
        pem: PEM,
      });
      await client.certificates.delete(c.key);
      const res = createTestClient().certificates.retrieve(c.key);
      await expect(res).rejects.toThrow(NotFoundError);
      await expect(res).rejects.toThrow(`Certificates with keys [${c.key}] not found`);
    });

    it("should drop its private key from the secrets cache", async () => {
      const client = createTestClient();
      const c = await client.certificates.create({
        name: "Cached key",
        source: "stored",
        pem: PEM,
        privateKey: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const { privateKey } = c;
      await client.secrets.retrieve(privateKey);
      await waitForStreamLive(client.connection);
      await createTestClient().certificates.delete(c.key);
      await expect
        .poll(async () => {
          try {
            await client.secrets.retrieve(privateKey);
            return false;
          } catch (e) {
            return NotFoundError.matches(e);
          }
        })
        .toBe(true);
    });

    it("should refuse to delete a private key that a certificate uses", async () => {
      const client = createTestClient();
      const c = await client.certificates.create({
        name: "In use",
        source: "stored",
        pem: PEM,
        privateKey: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const { privateKey } = c;
      const del = client.secrets.delete(privateKey);
      await expect(del).rejects.toThrow(ValidationError);
      await expect(del).rejects.toThrow(
        'cannot delete secret "In use private key" because certificate "In use" uses ' +
          "it: validation error",
      );
      const retrieved = await createTestClient().secrets.retrieve(privateKey);
      expect(retrieved.key).toBe(privateKey);
    });
  });

  describe("grant", () => {
    it("should grant a rack the private key of a certificate", async () => {
      const client = createTestClient();
      const r = await client.racks.create({ name: "Grant rack" });
      const c = await client.certificates.create({
        name: "Grant",
        source: "stored",
        pem: PEM,
        privateKey: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const grant = await client.secrets.grant({
        subject: rack.ontologyID(r.key),
        secrets: [c.privateKey],
      });
      expect(grant).toHaveLength(36);
    });
  });
});
