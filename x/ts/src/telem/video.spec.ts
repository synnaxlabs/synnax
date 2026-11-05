// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { decodeVideoFrame, encodeVideoFrame, type VideoFrame } from "@/telem/video";

const KEY: VideoFrame = {
  keyframe: true,
  codec: "h264",
  data: new Uint8Array([0, 0, 0, 1, 0x67]),
};
const DELTA: VideoFrame = {
  keyframe: false,
  codec: "h264",
  data: new Uint8Array([0, 0, 1, 0x41]),
};

describe("video", () => {
  describe("encodeVideoFrame", () => {
    it("should prefix the access unit with the flags and codec bytes", () => {
      expect(encodeVideoFrame(KEY)).toEqual(new Uint8Array([1, 1, 0, 0, 0, 1, 0x67]));
      expect(encodeVideoFrame(DELTA)).toEqual(new Uint8Array([0, 1, 0, 0, 1, 0x41]));
    });
  });

  describe("decodeVideoFrame", () => {
    it("should decode an encoded frame", () => {
      expect(decodeVideoFrame(encodeVideoFrame(KEY))).toEqual(KEY);
      expect(decodeVideoFrame(encodeVideoFrame(DELTA))).toEqual(DELTA);
    });

    it("should decode a frame with an empty access unit", () => {
      expect(decodeVideoFrame(new Uint8Array([0, 1])).data).toEqual(new Uint8Array());
    });

    it("should alias the sample", () => {
      const sample = encodeVideoFrame(KEY);
      expect(decodeVideoFrame(sample).data.buffer).toBe(sample.buffer);
    });

    it.each([
      [[1], "video sample has 1 bytes, but the header needs 2"],
      [[3, 1], "video sample has unknown flags 0x03"],
      [[1, 9], "video sample has unknown codec 9"],
    ])("should reject the malformed sample %j", (sample, message) => {
      expect(() => decodeVideoFrame(new Uint8Array(sample))).toThrow(message);
    });
  });
});
