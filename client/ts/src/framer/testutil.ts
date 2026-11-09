// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, TimeRange } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { Codec } from "@/framer/codec";
import { Frame, type Payload } from "@/framer/frame";

export const createKeys = (n: number): channel.Key[] =>
  Array.from({ length: n }, (_, i) => i + 1);

interface PayloadOptions {
  /** Base alignment for every series in the frame. */
  alignment?: bigint;
  /** Give every series a distinct time range and alignment, defeating the codec's
   * equal-props fast paths. */
  varied?: boolean;
}

/** Builds a frame payload with one float32 series per key. */
export const createPayload = (
  keys: channel.Key[],
  samplesPerChannel: number,
  { alignment = 0n, varied = false }: PayloadOptions = {},
): Payload => {
  const data = new Float32Array(samplesPerChannel);
  for (let i = 0; i < data.length; i++) data[i] = i;
  return {
    keys: [...keys],
    series: keys.map((_, i) => ({
      dataType: DataType.FLOAT32,
      data: data.buffer.slice(0),
      alignment: varied ? alignment + BigInt(i) : alignment,
      timeRange: varied ? new TimeRange(i * 1000, (i + 1) * 1000) : TimeRange.ZERO,
    })),
  };
};

/** Builds a sequence of frames with monotonically increasing alignments, the shape
 * the streaming run loop ingests in steady state. */
export const createSequence = (
  keys: channel.Key[],
  samplesPerChannel: number,
  count: number,
): Frame[] =>
  Array.from(
    { length: count },
    (_, i) =>
      new Frame(
        createPayload(keys, samplesPerChannel, {
          alignment: BigInt(i * samplesPerChannel),
        }),
      ),
  );

export const createCodec = (keys: channel.Key[]): Codec =>
  new Codec(
    keys,
    keys.map(() => DataType.FLOAT32),
  );
