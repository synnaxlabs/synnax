// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { dimensions } from "@/cli/mp4";

const box = (type: string, ...payload: Buffer[]): Buffer => {
  const body = Buffer.concat(payload);
  const header = Buffer.alloc(8);
  header.writeUInt32BE(body.length + 8);
  header.write(type, 4, "latin1");
  return Buffer.concat([header, body]);
};

// A version 0 track header: 84 bytes, ending in the 16.16 width and height.
const tkhd = (width: number, height: number): Buffer => {
  const payload = Buffer.alloc(84);
  payload.writeUInt32BE(width << 16, 76);
  payload.writeUInt32BE(height << 16, 80);
  return box("tkhd", payload);
};

const mp4 = (...tracks: Buffer[]): Buffer =>
  Buffer.concat([
    box("ftyp", Buffer.from("isom")),
    box("moov", box("mvhd", Buffer.alloc(100)), ...tracks.map((t) => box("trak", t))),
    box("mdat", Buffer.alloc(16)),
  ]);

describe("cli.mp4", () => {
  describe("dimensions", () => {
    it("should read the size of the video track", () => {
      expect(dimensions(mp4(tkhd(1920, 1080)))).toEqual({ width: 1920, height: 1080 });
    });

    it("should skip a track with no size", () => {
      expect(dimensions(mp4(tkhd(0, 0), tkhd(1720, 1080)))).toEqual({
        width: 1720,
        height: 1080,
      });
    });

    it("should throw for a file with no video track", () => {
      expect(() => dimensions(mp4(tkhd(0, 0)))).toThrow("MP4 file has no video track");
    });
  });
});
