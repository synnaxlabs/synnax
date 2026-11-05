// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** The encodings of the access unit in a VideoFrame. h264 is H.264 in Annex B format,
 * with no B-frames. */
export const VIDEO_CODECS = ["h264"] as const;
export type VideoCodec = (typeof VIDEO_CODECS)[number];

const CODEC_BYTES: Record<VideoCodec, number> = { h264: 1 };
const BYTE_CODECS = new Map<number, VideoCodec>(
  VIDEO_CODECS.map((c) => [CODEC_BYTES[c], c]),
);

const HEADER_SIZE = 2;
const KEYFRAME_FLAG = 1;

/** One encoded picture. It is a single sample of a video series. */
export interface VideoFrame {
  /** True when the frame decodes without the frames before it. A keyframe carries its
   * codec parameter sets. */
  keyframe: boolean;
  /** The encoding of data. */
  codec: VideoCodec;
  /** One access unit. */
  data: Uint8Array;
}

/** @returns the frame as a sample of a video series. */
export const encodeVideoFrame = ({ keyframe, codec, data }: VideoFrame): Uint8Array => {
  const sample = new Uint8Array(HEADER_SIZE + data.byteLength);
  sample[0] = keyframe ? KEYFRAME_FLAG : 0;
  sample[1] = CODEC_BYTES[codec];
  sample.set(data, HEADER_SIZE);
  return sample;
};

/**
 * @returns the frame that a sample of a video series holds. The data of the frame
 * aliases the sample.
 * @throws Error when the header is malformed or names an unknown codec.
 */
export const decodeVideoFrame = (sample: Uint8Array): VideoFrame => {
  if (sample.byteLength < HEADER_SIZE)
    throw new Error(
      `video sample has ${sample.byteLength} bytes, but the header needs ` +
        `${HEADER_SIZE}`,
    );
  const flags = sample[0];
  if ((flags & ~KEYFRAME_FLAG) !== 0)
    throw new Error(
      `video sample has unknown flags 0x${flags.toString(16).padStart(2, "0")}`,
    );
  const codec = BYTE_CODECS.get(sample[1]);
  if (codec == null) throw new Error(`video sample has unknown codec ${sample[1]}`);
  return {
    keyframe: flags === KEYFRAME_FLAG,
    codec,
    data: sample.subarray(HEADER_SIZE),
  };
};
