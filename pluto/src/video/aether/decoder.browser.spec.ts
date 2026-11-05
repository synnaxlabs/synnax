// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type VideoFrame as EncodedFrame } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";

import { Decoder, type DecoderProps } from "@/video/aether/decoder";
import { h264 } from "@/video/aether/h264";
import { videoTest } from "@/video/aether/testutil";

const { STREAM, SMALL } = videoTest;

describe("Decoder", () => {
  const pictures: VideoFrame[] = [];
  const errors: Error[] = [];
  let decoder: Decoder | null = null;

  afterEach(() => {
    decoder?.close();
    pictures.splice(0).forEach((p) => p.close());
    errors.splice(0);
  });

  const open = (props: Partial<DecoderProps> = {}): Decoder => {
    decoder = new Decoder({
      onPicture: (p) => pictures.push(p),
      onError: (e) => errors.push(e),
      ...props,
    });
    return decoder;
  };

  const decodeAt = (d: Decoder, frames: EncodedFrame[], indexes: number[]): void =>
    indexes.forEach((i) => d.decode(frames[i], BigInt(i)));

  it("should decode every frame of a stream in order", async () => {
    decodeAt(open(), STREAM, [0, 1, 2, 3, 4, 5, 6, 7]);
    await expect.poll(() => pictures.length).toEqual(8);
    expect(pictures.map((p) => p.timestamp)).toEqual([0, 1, 2, 3, 4, 5, 6, 7]);
    expect(pictures[0].displayWidth).toEqual(64);
    expect(errors).toEqual([]);
  });

  it("should skip frames up to the next keyframe after a gap", async () => {
    decodeAt(open(), STREAM, [0, 1, 3, 4, 5, 6, 7]);
    await expect.poll(() => pictures.length).toEqual(6);
    expect(errors).toEqual([]);
  });

  it("should skip delta frames until the first keyframe", async () => {
    decodeAt(open(), STREAM, [1, 2, 3, 4, 5]);
    await expect.poll(() => pictures.length).toEqual(2);
  });

  it("should decode the new stream when the parameter sets change", async () => {
    const d = open();
    decodeAt(d, STREAM, [0, 1]);
    SMALL.forEach((f, i) => d.decode(f, BigInt(2 + i)));
    // WebKit drops the frames that the old stream still queues.
    await expect
      .poll(() => pictures.filter((p) => p.displayWidth === 32).length)
      .toEqual(2);
    expect(pictures.at(-1)?.displayWidth).toEqual(32);
    expect(errors).toEqual([]);
  });

  it("should report a keyframe with no SPS and wait for the next keyframe", async () => {
    const slices = h264.split(STREAM[0].data).filter((u) => (u[0] & 0x1f) === 5);
    const bare = new Uint8Array(slices.flatMap((u) => [0, 0, 0, 1, ...u]));
    const d = open();
    d.decode({ ...STREAM[0], data: bare }, 0n);
    decodeAt(d, STREAM, [1, 2, 3, 4, 5]);
    expect(errors.map((e) => e.message)).toEqual(["video keyframe has no SPS"]);
    await expect.poll(() => pictures.length).toEqual(2);
  });

  it("should drop frames to the next keyframe while the decoder falls behind", async () => {
    const d = open({
      createDecoder: (init) => {
        const real = new VideoDecoder(init);
        Object.defineProperty(real, "decodeQueueSize", { get: () => 100 });
        return real;
      },
    });
    decodeAt(d, STREAM, [0, 1, 2, 3, 4, 5, 6, 7]);
    await expect.poll(() => pictures.length).toEqual(2);
  });

  it("should decode no delta frames after it closes", async () => {
    const d = open();
    decodeAt(d, STREAM, [0]);
    await expect.poll(() => pictures.length).toEqual(1);
    d.close();
    decodeAt(d, STREAM, [1, 2, 3]);
    await new Promise((resolve) => setTimeout(resolve, 100));
    expect(pictures).toHaveLength(1);
    expect(errors).toEqual([]);
  });
});
