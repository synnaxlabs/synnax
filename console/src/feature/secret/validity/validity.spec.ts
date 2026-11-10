// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, type secret } from "@synnaxlabs/client";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Validity } from "@/feature/secret/validity";

// Noon UTC keeps the date the same in every time zone within eleven hours of UTC.
const NOON = TimeSpan.hours(12);
const NOT_BEFORE = new TimeStamp([2026, 1, 15]).add(NOON);
const NOT_AFTER = new TimeStamp([2026, 6, 15]).add(NOON);

const STORED: certificate.Certificate = {
  key: "8b0e6a6c-5c3e-4c58-9f3f-0d6f5a2a1b7e",
  name: "Server",
  source: "stored",
  pem: "",
  details: {
    subject: "CN=Server",
    issuer: "CN=Server",
    notBefore: NOT_BEFORE,
    notAfter: NOT_AFTER,
    sha1: "",
    sha256: "",
  },
};

const describeAt = (now: TimeStamp): Validity.Validity =>
  Validity.describeCertificate(STORED, now);

describe("secret/validity", () => {
  describe("describeCertificate", () => {
    it("warns with the start date before the certificate is valid", () => {
      expect(describeAt(NOT_BEFORE.sub(TimeSpan.SECOND))).toEqual({
        text: "Valid from January 15, 2026",
        status: "warning",
      });
    });

    it("reads as valid before the 30-day expiry window", () => {
      expect(describeAt(NOT_AFTER.sub(TimeSpan.days(31)))).toEqual({
        text: "Expires June 15, 2026",
      });
    });

    it("warns inside the 30-day expiry window", () => {
      expect(describeAt(NOT_AFTER.sub(TimeSpan.days(10)))).toEqual({
        text: "Expires June 15, 2026",
        status: "warning",
      });
    });

    it("warns exactly 30 days before expiry", () => {
      expect(describeAt(NOT_AFTER.sub(TimeSpan.days(30)))).toEqual({
        text: "Expires June 15, 2026",
        status: "warning",
      });
    });

    it("reports an error at the expiry instant", () => {
      expect(describeAt(NOT_AFTER)).toEqual({
        text: "Expired June 15, 2026",
        status: "error",
      });
    });

    it("reports an error after expiry", () => {
      expect(describeAt(NOT_AFTER.add(TimeSpan.days(1)))).toEqual({
        text: "Expired June 15, 2026",
        status: "error",
      });
    });

    it("shows the path of a file certificate with no details", () => {
      const c: certificate.Certificate = {
        key: "1f4f2c1e-8a8d-4b5e-9d3c-2a7b6c5d4e3f",
        name: "Client",
        source: "file",
        certificatePath: "/etc/ssl/client.pem",
      };
      expect(Validity.describeCertificate(c, NOT_AFTER)).toEqual({
        text: "/etc/ssl/client.pem",
      });
    });

    it("shows the expiry of a file certificate with details", () => {
      const c: certificate.Certificate = {
        key: "2a5f3d2e-9b9e-4c6f-8e4d-3b8c7d6e5f40",
        name: "Client",
        source: "file",
        certificatePath: "/etc/ssl/client.pem",
        details: STORED.details,
      };
      expect(Validity.describeCertificate(c, NOT_AFTER.add(TimeSpan.days(1)))).toEqual({
        text: "Expired June 15, 2026",
        status: "error",
      });
    });
  });

  describe("describeSecret", () => {
    it("shows when the value of a stored secret was set", () => {
      const s: secret.Secret = {
        key: "3c2b1a0f-9e8d-4c7b-8a69-5f4e3d2c1b0a",
        name: "Password",
        source: "stored",
        updated: new TimeStamp([2026, 5, 4]).add(NOON),
      };
      expect(Validity.describeSecret(s)).toBe("Set on May 4, 2026");
    });

    it("shows the file name of a file secret", () => {
      const s: secret.Secret = {
        key: "7a6b5c4d-3e2f-4a1b-9c8d-7e6f5a4b3c2d",
        name: "Token",
        source: "file",
        updated: NOT_BEFORE,
        fileName: "token",
      };
      expect(Validity.describeSecret(s)).toBe("Read from token");
    });
  });
});
