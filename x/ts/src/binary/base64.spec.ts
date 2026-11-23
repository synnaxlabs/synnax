// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { binary } from "@/binary";

describe("base64", () => {
  const bytes = new Uint8Array([0xfb, 0xff, 0xbf, 0x00, 0x41]);

  describe("encodeBase64", () => {
    it("should encode with the standard alphabet and padding", () => {
      expect(binary.encodeBase64(bytes)).toEqual("+/+/AEE=");
    });
    it("should encode with the base64url alphabet and no padding", () => {
      expect(binary.encodeBase64(bytes, { alphabet: "base64url" })).toEqual("-_-_AEE");
    });
    it("should encode no bytes as an empty string", () => {
      expect(binary.encodeBase64(new Uint8Array())).toEqual("");
    });
    it("should encode more bytes than one call can spread", () => {
      const large = new Uint8Array(200_000).map((_, i) => i % 256);
      expect(binary.decodeBase64(binary.encodeBase64(large))).toEqual(large);
    });
  });

  describe("decodeBase64", () => {
    it("should decode the standard alphabet", () => {
      expect(binary.decodeBase64("+/+/AEE=")).toEqual(bytes);
    });
    it("should decode the base64url alphabet without padding", () => {
      expect(binary.decodeBase64("-_-_AEE", { alphabet: "base64url" })).toEqual(bytes);
    });
    it("should throw on a string that is not base64", () => {
      expect(() => binary.decodeBase64("not base64!")).toThrow();
    });
  });
});
