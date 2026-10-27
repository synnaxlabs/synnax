// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schema } from "@/server/db";
import { channels, term } from "@/server/license/limits";
import { type Fact, type Notice } from "@/server/mail";

const inDays = (days: number): string => `${days} ${days === 1 ? "day" : "days"}`;

const facts = (lic: schema.License, org: schema.Organization): Fact[] => [
  { label: "License", value: lic.label },
  { label: "Organization", value: org.name },
  { label: "Term", value: term(lic) },
];

const base = (
  lic: schema.License,
  org: schema.Organization,
): Pick<Notice, "facts" | "action" | "reason"> => ({
  facts: facts(lic, org),
  action: { label: "View license", href: `/licenses/${lic.key}` },
  reason:
    org.kind === "team"
      ? `You get this email because you are an admin of ${org.name}.`
      : "You get this email because the license is yours.",
});

/** issued tells an organization that staff issued it a license. */
export const issued = (lic: schema.License, org: schema.Organization): Notice => {
  const notice = base(lic, org);
  return {
    subject: `Your Synnax license "${lic.label}" is ready`,
    heading: "Your new license is ready",
    paragraphs: [
      "Activate a machine in the portal to give a Core its license key.",
      `Any member of ${org.name} can activate machines, up to the license's seats.`,
    ],
    ...notice,
    facts: [
      ...notice.facts,
      { label: "Seats", value: String(lic.nodes) },
      { label: "Channels per Core", value: channels(lic.channels) },
    ],
  };
};

/** expiring warns that a license expires in `days` days. */
export const expiring = (
  lic: schema.License,
  org: schema.Organization,
  days: number,
): Notice => ({
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
export const revoked = (lic: schema.License, org: schema.Organization): Notice => ({
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
export const restored = (lic: schema.License, org: schema.Organization): Notice => ({
  subject: `Your Synnax license "${lic.label}" was restored`,
  heading: "Your license was restored",
  paragraphs: [
    "It can activate Cores again while its terms allow. Machines that held a seat " +
      "before the revocation still hold it.",
  ],
  ...base(lic, org),
});
