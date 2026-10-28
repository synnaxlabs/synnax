// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { thumbprint, toDER } from "@/feature/opcua/device/thumbprint";

const pem = (body: string): string =>
  `-----BEGIN CERTIFICATE-----\n${body}\n-----END CERTIFICATE-----\n`;

const CERTIFICATE = pem(
  [
    "MIIB9TCCAV6gAwIBAgIUPLROcCOcVDumNLLqyV+97pXHmTEwDQYJKoZIhvcNAQEL",
    "BQAwHTEbMBkGA1UEAwwSU3lubmF4IFRlc3QgU2VydmVyMB4XDTI2MTAyODA0Mjcx",
    "MloXDTI2MTAyOTA0MjcxMlowHTEbMBkGA1UEAwwSU3lubmF4IFRlc3QgU2VydmVy",
    "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC9lSYqhYGt1RvLtBEob3WPnBIF",
    "H1gGnbamdp6oTeT9T/ocgeGj6NwfBJCMvXtJACyt+4wQ3U58PSlHxInlytb0ZguQ",
    "wJUPWwJPt6QvSQlGpSYL4Aa0KPXo4U28NDCqzC8DIOHp9Wlx7oZRshcaq5tzuO9Q",
    "VroBAklBSn4b8rpAfwIDAQABozIwMDAdBgNVHQ4EFgQUnU/mKGIwrOfxubAFVpOu",
    "Or1kl1AwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOBgQAZJQuhHkgx",
    "kk3Ol4VOR7+MVZYaL2QqYolkYSVPj9M1Yx5LyM87D3VGxEgCVeqbJpaccAbxy/kF",
    "eqka2A/hEYYsnl+pXB1Ezc98iJWMW0dgffTITjQ6ShiU3QIcC/7TR+8V0liJefaM",
    "KmxBQ5NEX9x+LtU7W4pSSUN5pr96DYdSnw==",
  ].join("\n"),
);

describe("thumbprint", () => {
  it("should give the SHA-1 fingerprint that OpenSSL gives for a certificate", () => {
    expect(thumbprint(CERTIFICATE)).toBe("BFC239BF34332730CF5A296716BA7E65FECC4FEB");
  });

  it("should match the FIPS 180 test vectors", () => {
    expect(thumbprint(pem(""))).toBe("DA39A3EE5E6B4B0D3255BFEF95601890AFD80709");
    expect(thumbprint(pem("YWJj"))).toBe("A9993E364706816ABA3E25717850C26C9CD0D89D");
    expect(
      thumbprint(
        pem(
          "YWJjZGJjZGVjZGVmZGVmZ2VmZ2hmZ2hpZ2hpamhpamtpamtsamtsbWtsbW5sbW5vbW5vcG5vcHE=",
        ),
      ),
    ).toBe("84983E441C3BD26EBAAE4AA1F95129E5E54670F1");
  });

  it("should throw for a body that is not base64", () => {
    expect(() => thumbprint(pem("not base64!"))).toThrow();
  });
});

describe("toDER", () => {
  it("should decode the body without the armor and line breaks", () => {
    expect(Array.from(toDER(pem("YWJj")))).toEqual([0x61, 0x62, 0x63]);
  });
});
