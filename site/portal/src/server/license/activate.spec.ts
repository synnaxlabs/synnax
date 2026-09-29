// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq } from "drizzle-orm";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { activation, event, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { activate, decide, deny, reissue, release } from "@/server/license/activate";
import { type Signer } from "@/server/license/sign";
import {
  activationOf,
  HASH_A,
  HASH_B,
  HASH_C,
  LICENSE,
  NOW,
} from "@/server/license/testutil";
import {
  createActivation,
  createHarness,
  createLicense,
  createOrganization,
  readKey,
} from "@/testutil";

describe("activate.decide", () => {
  it("should grant a seat when one is free", () => {
    expect(
      decide({ license: LICENSE, activations: [], fingerprint: [HASH_A], now: NOW }),
    ).toEqual({ ok: true });
  });

  it("should keep the seat of a machine that shares any hash", () => {
    const held = activationOf([HASH_A, HASH_B]);
    const decision = decide({
      license: LICENSE,
      activations: [held],
      fingerprint: [HASH_B, HASH_C],
      now: NOW,
    });
    expect(decision).toEqual({ ok: true, existing: held });
  });

  it("should refuse a new machine when every seat is taken", () => {
    const decision = decide({
      license: LICENSE,
      activations: [activationOf([HASH_A]), activationOf([HASH_B])],
      fingerprint: [HASH_C],
      now: NOW,
    });
    expect(decision).toEqual({ ok: false, reason: "no_seats" });
  });

  it("should not count a released seat", () => {
    const decision = decide({
      license: LICENSE,
      activations: [
        activationOf([HASH_A]),
        activationOf([HASH_B], { releasedAt: NOW }),
      ],
      fingerprint: [HASH_C],
      now: NOW,
    });
    expect(decision).toEqual({ ok: true });
  });

  it("should let a released machine reactivate as a new seat", () => {
    const decision = decide({
      license: LICENSE,
      activations: [activationOf([HASH_A], { releasedAt: NOW })],
      fingerprint: [HASH_A],
      now: NOW,
    });
    expect(decision).toEqual({ ok: true });
  });

  it("should refuse a revoked license", () => {
    expect(
      decide({
        license: { ...LICENSE, revokedAt: NOW },
        activations: [],
        fingerprint: [HASH_A],
        now: NOW,
      }),
    ).toEqual({ ok: false, reason: "revoked" });
  });
});

describe("activate.deny", () => {
  it("should allow a license inside its term", () => {
    expect(deny(LICENSE, NOW)).toBeUndefined();
  });

  it("should refuse a revoked license", () => {
    expect(deny({ ...LICENSE, revokedAt: NOW }, NOW)).toBe("revoked");
  });

  it("should refuse an expired subscription without a fallback", () => {
    expect(deny({ ...LICENSE, expiresAt: new Date(NOW.getTime() - 1) }, NOW)).toBe(
      "expired",
    );
  });

  it("should allow an expired subscription that has a fallback", () => {
    expect(
      deny(
        { ...LICENSE, expiresAt: new Date(NOW.getTime() - 1), maxVersion: "0.60" },
        NOW,
      ),
    ).toBeUndefined();
  });
});

describe("activate", () => {
  let store: Memory;
  let signer: Signer;
  let org: Organization;
  beforeAll(async () => {
    store = await openMemory();
    signer = createHarness(store).portal.signer;
  });
  beforeEach(async () => {
    await store.clear();
    org = await createOrganization(store, { kind: "team", clerkOrgID: "org_a" });
  });

  const events = async () => await store.query.select().from(event);
  const activations = async () => await store.query.select().from(activation);

  describe("activate", () => {
    it("should grant a new machine a seat and sign its license key", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_A, HASH_B],
        name: "Test stand",
        actor: "user_a",
        now: NOW,
      });
      if (!result.ok) throw new Error(result.reason);
      expect(result.activation).toMatchObject({
        license: lic.key,
        fingerprint: [HASH_A, HASH_B],
        name: "Test stand",
        firstSeen: NOW,
        lastSeen: NOW,
        releasedAt: null,
      });
      expect(readKey(result.key)).toEqual({
        jti: lic.key,
        iat: Math.floor(NOW.getTime() / 1000),
        exp: Math.floor(lic.expiresAt!.getTime() / 1000),
        claims_version: 1,
        organization: org.key,
        edition: "e",
        fingerprints: [HASH_A, HASH_B],
        fingerprint_scheme: 1,
        machines: 2,
        channels: 0,
        required: [],
      });
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "activate",
          at: NOW,
          actor: "user_a",
          organization: org.key,
          license: lic.key,
          activation: result.activation.key,
          detail: { fingerprint: [HASH_A, HASH_B] },
        }),
      ]);
    });

    it("should carry the ceiling and cap of the license into the key", async () => {
      const lic = await createLicense(store, {
        organization: org.key,
        term: "perpetual",
        expiresAt: null,
        maxVersion: "0.62",
        channels: 500,
      });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_A],
        name: "Rig",
        actor: "user_a",
        now: NOW,
      });
      if (!result.ok) throw new Error(result.reason);
      const claims = readKey(result.key);
      expect(claims).not.toHaveProperty("exp");
      expect(claims).toMatchObject({ max_version: "0.62", channels: 500 });
    });

    it("should keep the seat of a machine that shares a host hash", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const held = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A, HASH_B],
        name: "Old name",
        firstSeen: new Date("2026-09-01T00:00:00Z"),
        lastSeen: new Date("2026-09-01T00:00:00Z"),
      });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_B, HASH_C],
        name: "New name",
        actor: "user_a",
        now: NOW,
      });
      if (!result.ok) throw new Error(result.reason);
      expect(result.activation).toMatchObject({
        key: held.key,
        fingerprint: [HASH_A, HASH_B],
        name: "New name",
        firstSeen: new Date("2026-09-01T00:00:00Z"),
        lastSeen: NOW,
      });
      expect(readKey(result.key).fingerprints).toEqual([HASH_B, HASH_C]);
      expect(await activations()).toHaveLength(1);
      expect(await events()).toEqual([
        expect.objectContaining({ kind: "activate", activation: held.key }),
      ]);
    });

    it("should refuse a new machine when every seat is held", async () => {
      const lic = await createLicense(store, { organization: org.key, nodes: 1 });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_B],
        name: "Rig",
        actor: "user_a",
        now: NOW,
      });
      expect(result).toEqual({ ok: false, reason: "no_seats" });
      expect(await activations()).toHaveLength(1);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "activate_denied",
          at: NOW,
          actor: "user_a",
          organization: org.key,
          license: lic.key,
          activation: null,
          detail: { reason: "no_seats", fingerprint: [HASH_B] },
        }),
      ]);
    });

    it("should give a released seat to a new machine", async () => {
      const lic = await createLicense(store, { organization: org.key, nodes: 1 });
      await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
        releasedAt: NOW,
      });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_B],
        name: "Rig",
        actor: "user_a",
        now: NOW,
      });
      expect(result.ok).toBe(true);
      expect(await activations()).toHaveLength(2);
    });

    it("should seat a released machine again as a new activation", async () => {
      const lic = await createLicense(store, { organization: org.key, nodes: 1 });
      const released = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
        releasedAt: NOW,
      });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_A],
        name: "Rig",
        actor: "user_a",
        now: NOW,
      });
      if (!result.ok) throw new Error(result.reason);
      expect(result.activation.key).not.toBe(released.key);
      expect(result.activation.releasedAt).toBeNull();
    });

    it("should refuse a revoked license and record the denial", async () => {
      const lic = await createLicense(store, { organization: org.key, revokedAt: NOW });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_A],
        name: "Rig",
        actor: "user_a",
        now: NOW,
      });
      expect(result).toEqual({ ok: false, reason: "revoked" });
      expect(await activations()).toHaveLength(0);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "activate_denied",
          at: NOW,
          detail: { reason: "revoked", fingerprint: [HASH_A] },
        }),
      ]);
    });

    it("should refuse an expired subscription without a fallback", async () => {
      const lic = await createLicense(store, {
        organization: org.key,
        expiresAt: new Date(NOW.getTime() - 1000),
      });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_A],
        name: "Rig",
        actor: "user_a",
        now: NOW,
      });
      expect(result).toEqual({ ok: false, reason: "expired" });
    });

    it("should seat a machine on an expired subscription with a fallback", async () => {
      const expiresAt = new Date(NOW.getTime() - 1000);
      const lic = await createLicense(store, {
        organization: org.key,
        expiresAt,
        maxVersion: "0.60",
      });
      const result = await activate(store, signer, {
        licenseKey: lic.key,
        fingerprint: [HASH_A],
        name: "Rig",
        actor: "user_a",
        now: NOW,
      });
      if (!result.ok) throw new Error(result.reason);
      expect(readKey(result.key)).toMatchObject({
        exp: Math.floor(expiresAt.getTime() / 1000),
        max_version: "0.60",
      });
    });

    it("should throw a 404 for an unknown license", async () => {
      await expect(
        activate(store, signer, {
          licenseKey: crypto.randomUUID(),
          fingerprint: [HASH_A],
          name: "Rig",
          actor: "user_a",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "License not found" });
      expect(await events()).toHaveLength(0);
    });

    it("should never seat more machines than the license allows", async () => {
      const lic = await createLicense(store, { organization: org.key, nodes: 2 });
      const results = await Promise.all(
        [HASH_A, HASH_B, HASH_C].map(
          async (hash) =>
            await activate(store, signer, {
              licenseKey: lic.key,
              fingerprint: [hash],
              name: "Rig",
              actor: "user_a",
              now: NOW,
            }),
        ),
      );
      expect(results.filter((r) => r.ok)).toHaveLength(2);
      expect(results.filter((r) => !r.ok)).toEqual([{ ok: false, reason: "no_seats" }]);
      expect(await activations()).toHaveLength(2);
    });
  });

  describe("reissue", () => {
    it("should sign a fresh key for a held seat and record the download", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const held = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
        lastSeen: new Date("2026-09-01T00:00:00Z"),
      });
      const result = await reissue(store, signer, {
        activationKey: held.key,
        actor: "user_a",
        now: NOW,
      });
      expect(result.license.key).toBe(lic.key);
      expect(readKey(result.key)).toMatchObject({
        jti: lic.key,
        fingerprints: [HASH_A],
        iat: Math.floor(NOW.getTime() / 1000),
      });
      const [row] = await store.query
        .select()
        .from(activation)
        .where(eq(activation.key, held.key));
      expect(row.lastSeen).toEqual(NOW);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "download",
          at: NOW,
          actor: "user_a",
          organization: org.key,
          license: lic.key,
          activation: held.key,
          detail: {},
        }),
      ]);
    });

    it("should throw a 404 for a released seat", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const released = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
        releasedAt: NOW,
      });
      await expect(
        reissue(store, signer, {
          activationKey: released.key,
          actor: "user_a",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "Activation not found" });
      expect(await events()).toHaveLength(0);
    });

    it("should throw a 404 for an unknown seat", async () => {
      await expect(
        reissue(store, signer, {
          activationKey: crypto.randomUUID(),
          actor: "user_a",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "Activation not found" });
    });

    it("should throw a 400 once the license is revoked", async () => {
      const lic = await createLicense(store, { organization: org.key, revokedAt: NOW });
      const held = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      await expect(
        reissue(store, signer, { activationKey: held.key, actor: "user_a", now: NOW }),
      ).rejects.toMatchObject({
        status: 400,
        message: "This license has been revoked.",
      });
      expect(await events()).toHaveLength(0);
    });

    it("should throw a 400 once the subscription has expired", async () => {
      const lic = await createLicense(store, {
        organization: org.key,
        expiresAt: new Date(NOW.getTime() - 1000),
      });
      const held = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      await expect(
        reissue(store, signer, { activationKey: held.key, actor: "user_a", now: NOW }),
      ).rejects.toMatchObject({ status: 400, message: "This license has expired." });
    });
  });

  describe("release", () => {
    it("should free the seat and record the release", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const held = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      await release(store, { activationKey: held.key, actor: "user_a", now: NOW });
      const [row] = await store.query
        .select()
        .from(activation)
        .where(eq(activation.key, held.key));
      expect(row.releasedAt).toEqual(NOW);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "release",
          at: NOW,
          actor: "user_a",
          organization: org.key,
          license: lic.key,
          activation: held.key,
          detail: {},
        }),
      ]);
    });

    it("should do nothing for a seat already released", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const releasedAt = new Date("2026-09-01T00:00:00Z");
      const held = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
        releasedAt,
      });
      await release(store, { activationKey: held.key, actor: "user_a", now: NOW });
      const [row] = await store.query
        .select()
        .from(activation)
        .where(eq(activation.key, held.key));
      expect(row.releasedAt).toEqual(releasedAt);
      expect(await events()).toHaveLength(0);
    });

    it("should throw a 404 for an unknown seat", async () => {
      await expect(
        release(store, {
          activationKey: crypto.randomUUID(),
          actor: "user_a",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "Activation not found" });
    });
  });
});
