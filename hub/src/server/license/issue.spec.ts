// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { changes, type IssueArgs, validate } from "@/server/license/issue";
import { LICENSE, NOW } from "@/server/license/testutil";

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
