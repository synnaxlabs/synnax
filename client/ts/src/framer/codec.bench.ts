// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bench, describe } from "vitest";

import { allocSuite, makeCodec, makeFramePayload, makeKeys } from "@/framer/benchutil";
import { Frame } from "@/framer/frame";

// SHAPES are [channels, samples per channel]. They bracket the realistic range of
// streamed frames: a few high-rate channels, wide dashboards, and control-state
// frames with one sample across many channels.
const SHAPES: [number, number][] = [
  [1, 1000],
  [10, 100],
  [100, 10],
  [1000, 1],
];

const label = ([c, s]: [number, number]): string => `${c}ch x ${s}smp`;

describe("encode", () => {
  for (const shape of SHAPES) {
    const keys = makeKeys(shape[0]);
    const codec = makeCodec(keys);
    const payload = makeFramePayload(keys, shape[1]);
    bench(label(shape), () => {
      codec.encode(payload);
    });
  }
});

// The real writer path re-materializes the payload from a Frame on every write.
describe("encode from frame", () => {
  for (const shape of SHAPES) {
    const keys = makeKeys(shape[0]);
    const codec = makeCodec(keys);
    const frame = new Frame(makeFramePayload(keys, shape[1]));
    bench(label(shape), () => {
      codec.encode(frame);
    });
  }
});

describe("encode varied", () => {
  const shape: [number, number] = [100, 10];
  const keys = makeKeys(shape[0]);
  const codec = makeCodec(keys);
  const payload = makeFramePayload(keys, shape[1], { varied: true });
  bench(label(shape), () => {
    codec.encode(payload);
  });
});

describe("decode", () => {
  for (const shape of SHAPES) {
    const keys = makeKeys(shape[0]);
    const codec = makeCodec(keys);
    const wire = codec.encode(makeFramePayload(keys, shape[1]));
    bench(label(shape), () => {
      codec.decode(wire);
    });
  }
});

describe("decode varied", () => {
  const shape: [number, number] = [100, 10];
  const keys = makeKeys(shape[0]);
  const codec = makeCodec(keys);
  const wire = codec.encode(makeFramePayload(keys, shape[1], { varied: true }));
  bench(label(shape), () => {
    codec.decode(wire);
  });
});

{
  const keys = makeKeys(100);
  const codec = makeCodec(keys);
  const payload = makeFramePayload(keys, 10);
  const frame = new Frame(makeFramePayload(keys, 10));
  const wire = codec.encode(makeFramePayload(keys, 10));
  allocSuite("codec 100ch x 10smp", [
    ["encode", () => void codec.encode(payload)],
    ["encode from frame", () => void codec.encode(frame)],
    ["decode", () => void codec.decode(wire)],
  ]);
}
