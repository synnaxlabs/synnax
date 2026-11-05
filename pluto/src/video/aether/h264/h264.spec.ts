// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { h264 } from "@/video/aether/h264";

const SPS = [0x67, 0x42, 0xc0, 0x1f, 0xda];
const PPS = [0x68, 0xce, 0x3c, 0x80];
const IDR = [0x65, 0x88, 0x84];

describe("h264", () => {
  describe("split", () => {
    it("should split units after 3-byte and 4-byte start codes", () => {
      const au = new Uint8Array([
        0,
        0,
        0,
        1,
        ...SPS,
        0,
        0,
        1,
        ...PPS,
        0,
        0,
        0,
        1,
        ...IDR,
      ]);
      expect(h264.split(au).map((u) => [...u])).toEqual([SPS, PPS, IDR]);
    });

    it("should drop trailing zero bytes from each unit", () => {
      const au = new Uint8Array([0, 0, 1, ...IDR, 0, 0]);
      expect(h264.split(au).map((u) => [...u])).toEqual([IDR]);
    });

    it("should return no units for data with no start code", () => {
      expect(h264.split(new Uint8Array(IDR))).toEqual([]);
    });
  });

  describe("spsOf", () => {
    it("should return the sequence parameter set among the units", () => {
      const sps = h264.spsOf([PPS, SPS, IDR].map((u) => new Uint8Array(u)));
      expect([...sps!]).toEqual(SPS);
    });

    it("should return undefined without a sequence parameter set", () => {
      expect(h264.spsOf([PPS, IDR].map((u) => new Uint8Array(u)))).toBeUndefined();
    });

    it("should return undefined for a parameter set too short to name a level", () => {
      expect(h264.spsOf([new Uint8Array([0x67, 0x42, 0xc0])])).toBeUndefined();
    });
  });

  describe("codecOf", () => {
    it("should name the profile, constraints, and level of the parameter set", () => {
      expect(h264.codecOf(new Uint8Array(SPS))).toEqual("avc1.42c01f");
    });
  });
});
