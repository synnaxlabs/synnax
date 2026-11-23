// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { certificate } from "@/certificate";
import { LINES } from "@/certificate/where";
import { matches } from "@/certificate/where.gen";

const stored = (privateKey?: string): certificate.Certificate =>
  certificate.certificateZ.parse({
    name: "Stored",
    source: "stored",
    pem: "",
    privateKey,
    details: {
      subject: "CN=a",
      issuer: "CN=a",
      notBefore: 0,
      notAfter: 0,
      sha1: "",
      sha256: "",
    },
  });

const file = (privateKeyFile?: string): certificate.Certificate =>
  certificate.certificateZ.parse({
    name: "File",
    source: "file",
    certificatePath: "/a.crt",
    privateKeyFile,
    rack: 1,
  });

describe("Where", () => {
  describe("keyed", () => {
    it.each([
      ["a stored certificate with a key", stored(uuid.create()), true],
      ["a stored certificate with no key", stored(), false],
      ["a file certificate with a key file", file("client.pem"), true],
      ["a file certificate with no key file", file(), false],
    ])("should match %s by whether it has a key", (_, c, keyed) => {
      expect(matches(LINES, c, { keyed })).toBe(true);
      expect(matches(LINES, c, { keyed: !keyed })).toBe(false);
    });
  });
});
