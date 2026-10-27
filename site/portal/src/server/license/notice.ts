// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type License, type Organization } from "@/server/db/schema";
import { type Fact, type Notice } from "@/server/mail";

const day = (d: Date): string => d.toISOString().slice(0, 10);

const inDays = (days: number): string => `${days} ${days === 1 ? "day" : "days"}`;

const facts = (lic: License, org: Organization): Fact[] => {
  const facts: Fact[] = [
    { label: "License", value: lic.label },
    { label: "Organization", value: org.name },
  ];
  if (lic.expiresAt != null)
    facts.push({ label: "Expires", value: day(lic.expiresAt) });
  return facts;
};

const base = (
  lic: License,
  org: Organization,
): Pick<Notice, "facts" | "action" | "reason"> => ({
  facts: facts(lic, org),
  action: { label: "View license", href: `/licenses/${lic.key}` },
  reason:
    org.kind === "team"
      ? `You get this email because you are an admin of ${org.name}.`
      : "You get this email because the license is yours.",
});

/** expiring warns that a license expires in `days` days. */
export const expiring = (lic: License, org: Organization, days: number): Notice => ({
  subject: `Your Synnax license expires in ${inDays(days)}`,
  heading: `Your license expires in ${inDays(days)}`,
  paragraphs: [
    lic.maxVersion == null
      ? "After it expires, Cores running under it refuse to start."
      : `After it expires, it covers Core versions up to ${lic.maxVersion}.`,
    "Reply to this email to renew it.",
  ],
  ...base(lic, org),
});

/** revoked tells an organization that staff revoked one of its licenses. */
export const revoked = (lic: License, org: Organization): Notice => ({
  subject: `Your Synnax license "${lic.label}" was revoked`,
  heading: "Your license was revoked",
  paragraphs: [
    "It can no longer activate a Core. A Core checks its license offline, so a Core " +
      "that already holds it keeps running as its terms allow, even after a restart.",
    "Reply to this email if you think this is a mistake.",
  ],
  ...base(lic, org),
});

/** restored tells an organization that staff restored one of its licenses. */
export const restored = (lic: License, org: Organization): Notice => ({
  subject: `Your Synnax license "${lic.label}" was restored`,
  heading: "Your license was restored",
  paragraphs: [
    "It can activate Cores again while its terms allow. Machines that held a seat " +
      "before the revocation still hold it.",
  ],
  ...base(lic, org),
});
