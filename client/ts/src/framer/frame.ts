// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Size, LazyArray, TimeRange, toArray, DataType, unique } from "@synnaxlabs/x";
import { z } from "zod";

import {
  ChannelKeyOrName,
  ChannelKeys,
  ChannelNames,
  ChannelParams,
} from "@/channel/payload";
import { UnexpectedError, ValidationError } from "@/errors";

export class Frame {
  readonly keys: ChannelKeyOrName[];
  readonly arrays: LazyArray[];

  constructor(
    data:
      | FramePayload
      | Map<ChannelKeyOrName, LazyArray[]>
      | LazyArray[]
      | LazyArray = [],
    keys: ChannelParams = []
  ) {
    this.keys = [];
    this.arrays = [];
    if (Array.isArray(data)) {
      const arrKeys = toArray(keys);
      if (arrKeys.length !== data.length)
        throw new ValidationError("keys and data must be the same length");
      data.forEach((d, i) => this.push(arrKeys[i], d));
    } else if ("keys" in data) {
      const v = data as FramePayload;
      if (v.arrays == null || v.keys == null || v.keys.length !== v.arrays.length)
        throw new ValidationError("arrays and keys must be defined");
      v.keys.forEach((key, i) =>
        this.push(key, arrayFromPayload((v.arrays as LazyArray[])[i]))
      );
    } else if (data instanceof LazyArray) this.push(keys as string, data);
    else
      (data as Map<ChannelKeyOrName, LazyArray[]>).forEach(
        (v: LazyArray[], k: ChannelKeyOrName) => this.push(k, ...v)
      );
  }

  private get keyVariant(): "key" | "name" | null {
    if (this.keys.length === 0) return null;
    const firstKey = this.keys[0];
    return typeof firstKey === "string" ? "name" : "key";
  }

  get channelKeys(): number[] {
    if (this.keyVariant !== "key") throw new UnexpectedError("keyVariant is not key");
    return this.keys as number[];
  }

  get channelNames(): string[] {
    if (this.keyVariant !== "name") throw new UnexpectedError("keyVariant is not name");
    return this.keys as string[];
  }

  toPayload(): FramePayload {
    return {
      arrays: this.arrays.map((a) => arrayToPayload(a)),
      keys: this.channelKeys,
    };
  }

  get isVertical(): boolean {
    return unique(this.keys).length === this.keys.length;
  }

  get isHorizontal(): boolean {
    return unique(this.keys).length === 1;
  }

  get isWeaklyAligned(): boolean {
    if (this.keys.length <= 1) return true;
    return this.timeRanges.every((tr) => tr.equals(this.timeRanges[0]));
  }

  timeRange(key?: ChannelKeyOrName): TimeRange {
    if (key == null) {
      if (this.keys.length === 0) return TimeRange.ZERO;
      const start = Math.min(...this.arrays.map((a) => a.timeRange.start.valueOf()));
      const end = Math.max(...this.arrays.map((a) => a.timeRange.end.valueOf()));
      return new TimeRange(start, end);
    }
    const group = this.get(key);
    if (group == null) return TimeRange.ZERO;
    return new TimeRange(
      group[0].timeRange.start,
      group[group.length - 1].timeRange.end
    );
  }

  get timeRanges(): TimeRange[] {
    return this.keys.map((key) => this.timeRange(key));
  }

  get(key: ChannelKeyOrName): LazyArray[];

  get(keys: ChannelKeys | ChannelNames): Frame;

  /**
   * @returns all typed arrays matching the given key. If the frame is vertical,
   * this will return an array of length 1. If the frame is horiztonal, returns all
   * arrays in the frame.
   */
  get(key: ChannelKeyOrName | ChannelKeys | ChannelNames): LazyArray[] | Frame {
    // @ts-expect-error
    if (Array.isArray(key)) return this.filter((k) => key.includes(k));
    return this.arrays.filter((_, i) => this.keys[i] === key);
  }

  push(key: ChannelKeyOrName, ...v: LazyArray[]): void;

  push(frame: Frame): void;

  push(keyOrFrame: ChannelKeyOrName | Frame, ...v: LazyArray[]): void {
    if (keyOrFrame instanceof Frame) {
      this.arrays.push(...keyOrFrame.arrays);
      this.keys.push(...keyOrFrame.keys);
    } else {
      this.arrays.push(...v);
      this.keys.push(...Array.from({ length: v.length }, () => keyOrFrame));
    }
  }

  /**
   * @returns a shallow copy of this frame containing all typed arrays in the current frame and the
   * provided frame.
   */
  concat(frame: Frame): Frame {
    return new Frame([...this.arrays, ...frame.arrays], [...this.keys, ...frame.keys]);
  }

  has(channel: ChannelKeyOrName): boolean {
    return this.keys.includes(channel);
  }

  map(
    fn: (
      k: ChannelKeyOrName,
      arr: LazyArray,
      i: number
    ) => [ChannelKeyOrName, LazyArray]
  ): Frame {
    const frame = new Frame();
    this.forEach((k, arr, i) => frame.push(...fn(k, arr, i)));
    return frame;
  }

  forEach(fn: (k: ChannelKeyOrName, arr: LazyArray, i: number) => void): void {
    this.keys.forEach((k, i) => {
      const a = this.arrays[i];
      fn(k, a, i);
    });
  }

  filter(fn: (k: ChannelKeyOrName, arr: LazyArray, i: number) => boolean): Frame {
    const frame = new Frame();
    this.keys.forEach((k, i) => {
      const a = this.arrays[i];
      if (fn(k, a, i)) frame.push(k, a);
    });
    return frame;
  }

  get size(): Size {
    return new Size(
      this.arrays.reduce((acc, v) => acc.add(new Size(v.buffer.byteLength)), Size.ZERO)
    );
  }
}

export const array = z.object({
  timeRange: TimeRange.z.optional(),
  dataType: DataType.z,
  data: z.string().transform(
    (s) =>
      new Uint8Array(
        atob(s)
          .split("")
          .map((c) => c.charCodeAt(0))
      ).buffer
  ),
});

export type ArrayPayload = z.infer<typeof array>;

export const frameZ = z.object({
  keys: z.number().array().nullable().default([]),
  arrays: array.array().nullable().default([]),
});

export type FramePayload = z.infer<typeof frameZ>;

export const arrayFromPayload = (payload: ArrayPayload): LazyArray => {
  const { dataType, data, timeRange } = payload;
  return new LazyArray(data, dataType, timeRange);
};

export const arrayToPayload = (array: LazyArray): ArrayPayload => {
  return {
    timeRange: array._timeRange,
    dataType: array.dataType,
    data: new Uint8Array(array.data.buffer),
  };
};
