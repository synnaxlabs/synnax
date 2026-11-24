// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

interface Dimensions {
  width: number;
  height: number;
}

interface Box {
  type: string;
  /** Offset of the box's payload. */
  start: number;
  end: number;
}

const children = (file: Buffer, start: number, end: number): Box[] => {
  const boxes: Box[] = [];
  for (let at = start; at + 8 <= end;) {
    let size = file.readUInt32BE(at);
    let header = 8;
    if (size === 1) {
      size = Number(file.readBigUInt64BE(at + 8));
      header = 16;
    } else if (size === 0) size = end - at;
    boxes.push({
      type: file.toString("latin1", at + 4, at + 8),
      start: at + header,
      end: at + size,
    });
    at += size;
  }
  return boxes;
};

/**
 * Reads the display size of the first video track of an MP4 file from its track
 * header.
 * @throws {Error} if the file has no track with a size.
 */
export const dimensions = (file: Buffer): Dimensions => {
  const moov = children(file, 0, file.length).find(({ type }) => type === "moov");
  for (const trak of children(file, moov?.start ?? 0, moov?.end ?? 0)) {
    if (trak.type !== "trak") continue;
    const tkhd = children(file, trak.start, trak.end).find(
      ({ type }) => type === "tkhd",
    );
    if (tkhd == null) continue;
    // The track header ends with the width and height in 16.16 fixed point.
    const width = file.readUInt32BE(tkhd.end - 8) >>> 16;
    const height = file.readUInt32BE(tkhd.end - 4) >>> 16;
    if (width > 0 && height > 0) return { width, height };
  }
  throw new Error("MP4 file has no video track");
};
