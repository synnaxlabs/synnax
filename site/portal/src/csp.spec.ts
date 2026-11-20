// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { hash, LOADER } from "@synnaxlabs/site-common/csp";
import { describe, expect, it } from "vitest";

import { csp } from "@/csp";

describe("csp", () => {
  it("should forbid other sites to frame the portal", () => {
    expect(csp().directives).toContain("frame-ancestors 'none'");
  });

  it("should run only scripts from the portal, Clerk, Turnstile, and PostHog", () => {
    expect(csp().scriptDirective).toEqual({
      resources: [
        "'self'",
        "https://*.clerk.accounts.dev",
        "https://clerk.synnaxlabs.com",
        "https://challenges.cloudflare.com",
        "https://t.synnaxlabs.com",
      ],
      hashes: [hash(LOADER)],
    });
  });

  it("should allow inline style attributes but not style elements", () => {
    expect(csp().styleDirective?.resources).toEqual([
      "'self'",
      { resource: "'unsafe-inline'", kind: "attribute" },
    ]);
  });

  it("should allow the Vercel toolbar only on a preview", () => {
    expect(JSON.stringify(csp("production"))).not.toContain("vercel.live");
    expect(csp("preview").scriptDirective?.resources).toContain("https://vercel.live");
  });
});
