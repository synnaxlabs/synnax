// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { type Event, event, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { dueNotice, sweep } from "@/server/license/expiry";
import { LICENSE, NOW } from "@/server/license/testutil";
import { type Mailer, memory, type Message } from "@/server/mail";
import { createLicense, createOrganization } from "@/testutil";

const DAY = 24 * 60 * 60 * 1000;
const expiringIn = (days: number) => ({
  ...LICENSE,
  expiresAt: new Date(NOW.getTime() + days * DAY),
});
const notice = (days: number): Event => ({
  key: days,
  at: NOW,
  kind: "expiry_notice",
  actor: "system",
  organization: LICENSE.organization,
  license: LICENSE.key,
  activation: null,
  detail: { days },
});

describe("expiry.dueNotice", () => {
  it("should be silent far from expiry", () => {
    expect(dueNotice(expiringIn(45), [], NOW)).toBeNull();
  });

  it("should send the 30 day warning inside its window once", () => {
    expect(dueNotice(expiringIn(29.5), [], NOW)).toBe(30);
    expect(dueNotice(expiringIn(29.5), [notice(30)], NOW)).toBeNull();
  });

  it("should send the most urgent unsent warning", () => {
    expect(dueNotice(expiringIn(6), [notice(30)], NOW)).toBe(7);
    expect(dueNotice(expiringIn(0.5), [notice(30), notice(7)], NOW)).toBe(1);
  });

  it("should skip an earlier window a later one has overtaken", () => {
    expect(dueNotice(expiringIn(6), [notice(7)], NOW)).toBeNull();
  });

  it("should be silent once expired, revoked, or perpetual", () => {
    expect(dueNotice(expiringIn(-1), [], NOW)).toBeNull();
    expect(dueNotice({ ...expiringIn(5), revokedAt: NOW }, [], NOW)).toBeNull();
    expect(dueNotice({ ...LICENSE, expiresAt: null }, [], NOW)).toBeNull();
  });
});

describe("expiry.sweep", () => {
  let store: Memory;
  let org: Organization;
  let mail: Mailer & { sent: Message[] };
  let asked: string[];
  const TO = ["admin@acme.com"];
  const recipients = async (key: string) => {
    asked.push(key);
    return TO;
  };
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    org = await createOrganization(store, {
      kind: "team",
      name: "Acme",
      clerkOrgID: "org_a",
    });
    mail = memory();
    asked = [];
  });

  const events = async () => await store.query.select().from(event);
  const createExpiring = async (days: number, values = {}) =>
    await createLicense(store, {
      organization: org.key,
      expiresAt: new Date(NOW.getTime() + days * DAY),
      ...values,
    });

  it("should mail a due warning and record it", async () => {
    const lic = await createExpiring(29.5, { label: "Site A" });
    const sent = await sweep({ store, mail, recipients, now: NOW });
    expect(sent).toEqual([{ license: lic.key, days: 30, to: TO }]);
    expect(asked).toEqual([org.key]);
    expect(mail.sent).toEqual([
      {
        to: TO,
        subject: "Your Synnax license expires in 30 days",
        text: [
          'The Synnax license "Site A" for Acme expires in 30 days, on ' +
            `${lic.expiresAt?.toISOString().slice(0, 10)}.`,
          "",
          "Cores running under it will refuse to start after that date.",
          "",
          "Reply to this email to renew.",
        ].join("\n"),
      },
    ]);
    expect(await events()).toEqual([
      expect.objectContaining({
        kind: "expiry_notice",
        actor: "system",
        organization: org.key,
        license: lic.key,
        activation: null,
        detail: { days: 30, to: TO },
      }),
    ]);
  });

  it("should word a last-day warning for a license with a fallback", async () => {
    await createExpiring(0.5, { maxVersion: "0.60" });
    await sweep({ store, mail, recipients, now: NOW });
    expect(mail.sent).toHaveLength(1);
    expect(mail.sent[0].subject).toBe("Your Synnax license expires in 1 day");
    expect(mail.sent[0].text).toContain("expires in 1 day, on");
    expect(mail.sent[0].text).toContain(
      "After that date it covers Core versions up to 0.60.",
    );
  });

  it("should send each warning once across sweeps", async () => {
    await createExpiring(29.5);
    await sweep({ store, mail, recipients, now: NOW });
    expect(await sweep({ store, mail, recipients, now: NOW })).toEqual([]);
    expect(mail.sent).toHaveLength(1);
    expect(await events()).toHaveLength(1);
  });

  it("should send each window as the expiry nears", async () => {
    const lic = await createExpiring(29.5);
    const at = (days: number) => new Date(NOW.getTime() + days * DAY);
    const days = [];
    for (const now of [NOW, at(23), at(29)]) {
      const sent = await sweep({ store, mail, recipients, now });
      days.push(...sent.map((s) => s.days));
    }
    expect(days).toEqual([30, 7, 1]);
    expect((await events()).map((e) => e.license)).toEqual([lic.key, lic.key, lic.key]);
  });

  it("should skip an organization with nobody to mail and retry it later", async () => {
    await createExpiring(29.5);
    const nobody = async () => [];
    expect(await sweep({ store, mail, recipients: nobody, now: NOW })).toEqual([]);
    expect(mail.sent).toHaveLength(0);
    expect(await events()).toHaveLength(0);
    expect(await sweep({ store, mail, recipients, now: NOW })).toHaveLength(1);
  });

  it("should leave the warning due when the mail fails", async () => {
    await createExpiring(29.5);
    const failing: Mailer = {
      send: async () => {
        throw new Error("mail: down");
      },
    };
    await sweep({ store, mail: failing, recipients, now: NOW }).catch(() => {});
    expect(await events()).toHaveLength(0);
    expect(await sweep({ store, mail, recipients, now: NOW })).toHaveLength(1);
    expect(mail.sent).toHaveLength(1);
  });

  it("should skip far, expired, revoked, and perpetual licenses", async () => {
    await createExpiring(45);
    await createExpiring(-1);
    await createExpiring(5, { revokedAt: NOW });
    await createLicense(store, {
      organization: org.key,
      term: "perpetual",
      expiresAt: null,
      maxVersion: "0.62",
    });
    expect(await sweep({ store, mail, recipients, now: NOW })).toEqual([]);
    expect(asked).toEqual([]);
    expect(mail.sent).toHaveLength(0);
    expect(await events()).toHaveLength(0);
  });

  it("should warn each organization's licenses to its own recipients", async () => {
    const other = await createOrganization(store, {
      kind: "team",
      name: "Globex",
      clerkOrgID: "org_b",
    });
    await createExpiring(6);
    await createLicense(store, {
      organization: other.key,
      expiresAt: new Date(NOW.getTime() + 6 * DAY),
    });
    const byOrg = async (key: string) => [`${key}@example.com`];
    const sent = await sweep({ store, mail, recipients: byOrg, now: NOW });
    expect(sent.map((s) => s.to).sort()).toEqual(
      [[`${org.key}@example.com`], [`${other.key}@example.com`]].sort(),
    );
    expect(sent.map((s) => s.days)).toEqual([7, 7]);
  });
});
