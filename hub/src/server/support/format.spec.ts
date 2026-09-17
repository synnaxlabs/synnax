// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { LICENSE, NOW } from "@/server/license/testutil";
import { comment, description, replyMail } from "@/server/support/format";

const ORG = {
  key: LICENSE.organization,
  kind: "team" as const,
  name: "Streeling",
  clerkOrgID: "org_1",
  linearCustomerID: null,
  ownerUserID: null,
  createdAt: NOW,
};
const SITE = "https://docs.synnaxlabs.com";

describe("format.description", () => {
  it("should link the organization and list its licenses", () => {
    const text = description({
      author: "Gaal Dornik",
      contact: "gaal@streeling.edu",
      text: "The Core will not start.",
      site: SITE,
      organization: ORG,
      licenses: [LICENSE, { ...LICENSE, label: "Old", revokedAt: NOW }],
      now: NOW,
    });
    expect(text).toContain(
      "Opened by Gaal Dornik (gaal@streeling.edu) from the portal.",
    );
    expect(text).toContain(`[Streeling](${SITE}/licenses?org=${ORG.key})`);
    expect(text).toContain("- Test rig: enterprise, 2 nodes, until 2027-03-01, active");
    expect(text).toContain("- Old: enterprise, 2 nodes, until 2027-03-01, revoked");
    expect(text.endsWith("---\n\nThe Core will not start.")).toBe(true);
  });

  it("should note the page of an anonymous feedback submission", () => {
    const text = description({
      author: "Anonymous visitor",
      contact: "",
      text: "typo",
      site: SITE,
      organization: null,
      licenses: [],
      page: "/reference/installation",
      now: NOW,
    });
    expect(text).toContain("Opened by Anonymous visitor from the portal.");
    expect(text).toContain(`Page: ${SITE}/reference/installation`);
    expect(text).not.toContain("Organization:");
    expect(text).not.toContain("Licenses:");
  });
});

describe("format.comment", () => {
  it("should attribute the message to its sender", () => {
    expect(comment("staff", "Hari", "Try again.")).toBe(
      "**Hari** (staff, via the portal)\n\nTry again.",
    );
  });
});

describe("format.replyMail", () => {
  it("should quote the reply and link back to the thread", () => {
    const mail = replyMail({
      title: "Core will not start",
      author: "Hari",
      text: "Try again.",
      url: `${SITE}/support/abc`,
    });
    expect(mail.subject).toBe("Re: Core will not start");
    expect(mail.text).toContain('Hari from Synnax replied to "Core will not start"');
    expect(mail.text).toContain(`Reply at ${SITE}/support/abc`);
  });
});
