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
  it("should apply the defaults", () => {
    expect(tls.tlsZ.parse({})).toEqual({ certificateValidationDisabled: false });
  });

  it("should parse both certificates", () => {
    expect(
      tls.tlsZ.parse({
        caCertificate: CERTIFICATE,
        clientCertificate: CERTIFICATE,
        certificateValidationDisabled: true,
      }),
    ).toEqual({
      caCertificate: CERTIFICATE,
      clientCertificate: CERTIFICATE,
      certificateValidationDisabled: true,
    });
  });

  it("should reject a certificate that is not a key", () => {
    expect(tls.tlsZ.safeParse({ caCertificate: "ca.pem" }).success).toBe(false);
  });
});
