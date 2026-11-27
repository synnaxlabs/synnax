// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { tls } from "@/tls";

const CERTIFICATE = "33333333-3333-4333-8333-333333333333";

describe("tlsZ", () => {
  it("should trust the system roots by default", () => {
    expect(tls.tlsZ.parse({})).toEqual({ serverTrust: { type: "system_roots" } });
  });

  it("should parse a certificate authority and a client certificate", () => {
    const value = {
      serverTrust: { type: "certificate_authority", certificate: CERTIFICATE },
      clientCertificate: CERTIFICATE,
    };
    expect(tls.tlsZ.parse(value)).toEqual(value);
  });

  it("should parse disabled validation", () => {
    const value = { serverTrust: { type: "validation_disabled" } };
    expect(tls.tlsZ.parse(value)).toEqual(value);
  });

  it("should reject a certificate that is not a key", () => {
    expect(
      tls.tlsZ.safeParse({
        serverTrust: { type: "certificate_authority", certificate: "ca.pem" },
      }).success,
    ).toBe(false);
  });
});
