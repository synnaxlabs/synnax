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

import { event, license, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import {
  amend,
  changes,
  floating,
  issue,
  type IssueArgs,
  revoke,
  validate,
} from "@/server/license/issue";
import { type Signer } from "@/server/license/sign";
import { HASH_A, HASH_B, LICENSE, NOW } from "@/server/license/testutil";
import {
  createActivation,
  createHarness,
  createLicense,
  createOrganization,
  readKey,
} from "@/testutil";

const base: IssueArgs = {
  organization: LICENSE.organization,
  edition: "enterprise",
  term: "subscription",
  nodes: 1,
  channels: 0,
  label: "Test",
  expiresAt: new Date("2027-03-01T00:00:00Z"),
  actor: "user_staff",
  now: NOW,
};

describe("issue.validate", () => {
  it("should accept a subscription with an expiry", () => {
    expect(() => validate(base)).not.toThrow();
  });

  it("should accept a subscription with a fallback ceiling", () => {
    expect(() => validate({ ...base, maxVersion: "0.62" })).not.toThrow();
  });

  it("should require an expiry in the future on a subscription", () => {
    expect(() => validate({ ...base, expiresAt: undefined })).toThrow(
      "A subscription needs an expiry",
    );
    expect(() => validate({ ...base, expiresAt: NOW })).toThrow(
      "The expiry must be in the future",
    );
  });

  it("should reject an expiry that is not a valid date", () => {
    expect(() =>
      validate({ ...base, expiresAt: new Date("not-a-dateT00:00:00Z") }),
    ).toThrow("The expiry must be a valid date");
  });

  it("should require a ceiling and forbid an expiry on a perpetual license", () => {
    expect(() =>
      validate({
        ...base,
        term: "perpetual",
        expiresAt: undefined,
        maxVersion: "0.62",
      }),
    ).not.toThrow();
    expect(() =>
      validate({ ...base, term: "perpetual", expiresAt: undefined }),
    ).toThrow("A perpetual license needs a maximum version");
    expect(() => validate({ ...base, term: "perpetual", maxVersion: "0.62" })).toThrow(
      "A perpetual license has no expiry",
    );
  });

  it("should reject a malformed ceiling and bad counts", () => {
    expect(() => validate({ ...base, maxVersion: "v0.62.1" })).toThrow(
      'Maximum version must look like "0.62"',
    );
    expect(() => validate({ ...base, nodes: 0 })).toThrow("Nodes must be");
    expect(() => validate({ ...base, channels: -1 })).toThrow("Channels must be");
  });

  it("should require a label", () => {
    expect(() => validate({ ...base, label: "" })).toThrow("Give the license a label");
  });
});

describe("issue.changes", () => {
  it("should list nothing when the terms are the same", () => {
    expect(changes(LICENSE, { ...LICENSE })).toEqual({});
  });

  it("should list each changed field as its before and after", () => {
    expect(changes(LICENSE, { ...LICENSE, nodes: 5, label: "Site B" })).toEqual({
      nodes: { from: 2, to: 5 },
      label: { from: LICENSE.label, to: "Site B" },
    });
  });

  it("should read an expiry as a date the log can print", () => {
    const later = new Date("2028-01-01T00:00:00Z");
    expect(changes(LICENSE, { ...LICENSE, expiresAt: later })).toEqual({
      expiresAt: {
        from: LICENSE.expiresAt?.toISOString(),
        to: later.toISOString(),
      },
    });
  });

  it("should ignore fields an amendment cannot alter", () => {
    expect(changes(LICENSE, { ...LICENSE, revokedAt: NOW })).toEqual({});
  });
});

describe("issue", () => {
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
  const stored = async (key: string) => {
    const [row] = await store.query.select().from(license).where(eq(license.key, key));
    return row;
  };

  describe("issue", () => {
    it("should record a subscription and its issue event", async () => {
      const lic = await issue(store, {
        ...base,
        organization: org.key,
        nodes: 3,
        channels: 100,
        label: "Site A",
        maxVersion: "0.62",
      });
      expect(lic).toMatchObject({
        organization: org.key,
        edition: "enterprise",
        term: "subscription",
        nodes: 3,
        channels: 100,
        label: "Site A",
        expiresAt: base.expiresAt,
        maxVersion: "0.62",
        issuedBy: "user_staff",
        issuedAt: NOW,
        revokedAt: null,
      });
      expect(await stored(lic.key)).toEqual(lic);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "issue",
          at: NOW,
          actor: "user_staff",
          organization: org.key,
          license: lic.key,
          activation: null,
          detail: { nodes: 3, channels: 100, term: "subscription" },
        }),
      ]);
    });

    it("should record a perpetual license with no expiry", async () => {
      const lic = await issue(store, {
        ...base,
        organization: org.key,
        term: "perpetual",
        expiresAt: undefined,
        maxVersion: "0.62",
      });
      expect(lic).toMatchObject({
        term: "perpetual",
        expiresAt: null,
        maxVersion: "0.62",
      });
    });

    it("should record nothing when the terms are invalid", async () => {
      await expect(
        issue(store, { ...base, organization: org.key, nodes: 0 }),
      ).rejects.toMatchObject({
        status: 400,
        message: "Nodes must be a whole number of at least 1",
      });
      expect(await store.query.select().from(license)).toHaveLength(0);
      expect(await events()).toHaveLength(0);
    });
  });

  describe("amend", () => {
    it("should change the terms, keep the key, and record what changed", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const later = new Date("2028-01-01T00:00:00Z");
      const after = await amend(store, {
        licenseKey: lic.key,
        term: "subscription",
        nodes: 5,
        channels: 10,
        label: "Site B",
        expiresAt: later,
        actor: "user_staff",
        now: NOW,
      });
      expect(after).toMatchObject({
        key: lic.key,
        nodes: 5,
        channels: 10,
        label: "Site B",
        expiresAt: later,
        maxVersion: null,
        issuedAt: lic.issuedAt,
      });
      expect(await stored(lic.key)).toEqual(after);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "amend",
          at: NOW,
          actor: "user_staff",
          organization: org.key,
          license: lic.key,
          detail: {
            nodes: { from: 2, to: 5 },
            channels: { from: 0, to: 10 },
            label: { from: "Test rig", to: "Site B" },
            expiresAt: {
              from: lic.expiresAt?.toISOString(),
              to: later.toISOString(),
            },
          },
        }),
      ]);
    });

    it("should turn a subscription perpetual, clearing its expiry", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const after = await amend(store, {
        licenseKey: lic.key,
        term: "perpetual",
        nodes: 2,
        channels: 0,
        label: "Test rig",
        maxVersion: "0.62",
        actor: "user_staff",
        now: NOW,
      });
      expect(after).toMatchObject({
        term: "perpetual",
        expiresAt: null,
        maxVersion: "0.62",
      });
      expect((await events())[0].detail).toEqual({
        term: { from: "subscription", to: "perpetual" },
        expiresAt: { from: lic.expiresAt?.toISOString(), to: null },
        maxVersion: { from: null, to: "0.62" },
      });
    });

    it("should refuse to drop the seats below the machines holding one", async () => {
      const lic = await createLicense(store, { organization: org.key, nodes: 3 });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_B] });
      await expect(
        amend(store, {
          licenseKey: lic.key,
          term: "subscription",
          nodes: 1,
          channels: 0,
          label: "Test rig",
          expiresAt: lic.expiresAt ?? undefined,
          actor: "user_staff",
          now: NOW,
        }),
      ).rejects.toMatchObject({
        status: 400,
        message: "2 machines hold a seat. Release one before lowering the limit to 1.",
      });
      expect((await stored(lic.key)).nodes).toBe(3);
      expect(await events()).toHaveLength(0);
    });

    it("should allow dropping the seats to the machines holding one", async () => {
      const lic = await createLicense(store, { organization: org.key, nodes: 3 });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_B],
        releasedAt: NOW,
      });
      const after = await amend(store, {
        licenseKey: lic.key,
        term: "subscription",
        nodes: 1,
        channels: 0,
        label: "Test rig",
        expiresAt: lic.expiresAt ?? undefined,
        actor: "user_staff",
        now: NOW,
      });
      expect(after.nodes).toBe(1);
    });

    it("should refuse to change a revoked license", async () => {
      const lic = await createLicense(store, { organization: org.key, revokedAt: NOW });
      await expect(
        amend(store, {
          licenseKey: lic.key,
          term: "subscription",
          nodes: 3,
          channels: 0,
          label: "Test rig",
          expiresAt: lic.expiresAt ?? undefined,
          actor: "user_staff",
          now: NOW,
        }),
      ).rejects.toMatchObject({
        status: 400,
        message: "A revoked license cannot be changed",
      });
      expect(await events()).toHaveLength(0);
    });

    it("should refuse invalid terms before touching the license", async () => {
      const lic = await createLicense(store, { organization: org.key });
      await expect(
        amend(store, {
          licenseKey: lic.key,
          term: "subscription",
          nodes: 2,
          channels: 0,
          label: "Test rig",
          actor: "user_staff",
          now: NOW,
        }),
      ).rejects.toMatchObject({
        status: 400,
        message: "A subscription needs an expiry",
      });
      expect(await stored(lic.key)).toEqual(lic);
    });

    it("should throw a 404 for an unknown license", async () => {
      await expect(
        amend(store, {
          licenseKey: crypto.randomUUID(),
          term: "subscription",
          nodes: 2,
          channels: 0,
          label: "Test rig",
          expiresAt: base.expiresAt,
          actor: "user_staff",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "License not found" });
    });
  });

  describe("revoke", () => {
    it("should stamp the revocation and record it", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const row = await revoke(store, {
        licenseKey: lic.key,
        actor: "user_staff",
        now: NOW,
      });
      expect(row).toMatchObject({ key: lic.key, revokedAt: NOW });
      expect(await stored(lic.key)).toEqual(row);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "revoke",
          at: NOW,
          actor: "user_staff",
          organization: org.key,
          license: lic.key,
          activation: null,
          detail: {},
        }),
      ]);
    });

    it("should throw a 404 for an unknown license", async () => {
      await expect(
        revoke(store, {
          licenseKey: crypto.randomUUID(),
          actor: "user_staff",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "License not found" });
      expect(await events()).toHaveLength(0);
    });

    it("should refuse a license already revoked, keeping the first stamp", async () => {
      const earlier = new Date(NOW.getTime() - 60_000);
      const lic = await createLicense(store, {
        organization: org.key,
        revokedAt: earlier,
      });
      await expect(
        revoke(store, { licenseKey: lic.key, actor: "user_staff", now: NOW }),
      ).rejects.toMatchObject({
        status: 400,
        message: "This license is already revoked",
      });
      expect((await stored(lic.key)).revokedAt).toEqual(earlier);
      expect(await events()).toHaveLength(0);
    });
  });

  describe("floating", () => {
    it("should sign a key bound to no machine and hold no seat", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const key = await floating(store, signer, {
        licenseKey: lic.key,
        actor: "user_staff",
        now: NOW,
      });
      expect(readKey(key)).toMatchObject({
        jti: lic.key,
        organization: org.key,
        fingerprints: [],
        machines: 2,
      });
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "download",
          at: NOW,
          actor: "user_staff",
          organization: org.key,
          license: lic.key,
          activation: null,
          detail: { floating: true },
        }),
      ]);
    });

    it("should sign for an expired subscription with a fallback", async () => {
      const lic = await createLicense(store, {
        organization: org.key,
        expiresAt: new Date(NOW.getTime() - 1000),
        maxVersion: "0.60",
      });
      const key = await floating(store, signer, {
        licenseKey: lic.key,
        actor: "user_staff",
        now: NOW,
      });
      expect(readKey(key)).toMatchObject({ max_version: "0.60" });
    });

    it("should refuse a revoked license", async () => {
      const lic = await createLicense(store, { organization: org.key, revokedAt: NOW });
      await expect(
        floating(store, signer, { licenseKey: lic.key, actor: "user_staff", now: NOW }),
      ).rejects.toMatchObject({
        status: 400,
        message: "This license has been revoked.",
      });
      expect(await events()).toHaveLength(0);
    });

    it("should refuse an expired subscription without a fallback", async () => {
      const lic = await createLicense(store, {
        organization: org.key,
        expiresAt: new Date(NOW.getTime() - 1000),
      });
      await expect(
        floating(store, signer, { licenseKey: lic.key, actor: "user_staff", now: NOW }),
      ).rejects.toMatchObject({ status: 400, message: "This license has expired." });
    });

    it("should throw a 404 for an unknown license", async () => {
      await expect(
        floating(store, signer, {
          licenseKey: crypto.randomUUID(),
          actor: "user_staff",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "License not found" });
    });
  });
});
