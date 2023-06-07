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
  ChannelKey,
  ChannelKeyOrName,
  ChannelKeys,
  ChannelNames,
  ChannelParams,
} from "@/channel/payload";
import { UnexpectedError, ValidationError } from "@/errors";

type LabeledBy = "key" | "name" | null;

const labeledBy = (labels: ChannelParams): LabeledBy => {
  const arrKeys = toArray(labels);
  if (arrKeys.length === 0) return null;
  if (typeof arrKeys[0] === "number") return "key";
  return "name";
};

const validateMatchedLabelsAndArrays = (
  labels: ChannelParams,
  arrays: LazyArray[]
): void => {
  const labelsArr = toArray(labels);
  if (labelsArr.length === arrays.length) return;
  const labeledBy_ = labeledBy(labels);
  if (labeledBy === null)
    throw new ValidationError(
      "[Frame] - channel keys or names must be provided when constructing a frame."
    );
  throw new ValidationError(
    `[Frame] - ${labeledBy_ as string}s and arrays must be the same length.
    Got ${labelsArr.length} ${labeledBy_ as string}s and ${arrays.length} arrays.`
  );
};

/**
 * A frame is a collection of related typed arrays keyed to a particular channel. Frames
 * can be keyed by channel name or channel key, but not both.
 *
 * Frames have two important characteristics: alignment and orientation.
 *
 * A frame's alignment defines how correlated the arrays for different channels in the
 * frame are:
 *
 * - A frame is weakly aligned if it meets the time range occupied by all arrays of a
 * particular channel is the same for all channels in the frame. This means that the
 * arrays for a particular channel can have gaps betwen them.
 *
 * - A strongly aligned frame means that all channels share the same rate/index and
 * there are no gaps in time between arrays. Strongly aligned frames are natural
 * to interpret, as the values in a particular 'row' of the frame share the same
 * timestamp. All frames written to Synnax must be strongly aligned.
 *
 * - Unaligned frames don't meet the requirements for weakly or strongly aligned frames.
 * Unaligned frames are common when reading from channels that don't have the same
 * index/rate and are continuous in different periods of time.
 *
 * Frames also have an orientation: horizontal, vertical, or square.
 *
 * - Horizontal frames have a single channel, and are strongly aligned by default.
 * A horizontal frame typically has a single array (in which case, it's also 'square'),
 * although it can have multiple arrays if all the arrays are continuous in time.
 *
 * - Vertical frames are strongly aligned and have on or more channels, but ONLY a single
 * array per channel. Synnax requires that all frames written to the database are
 * vertical.
 *
 * - Square frames are both horizontal and vertical. Only a frame with a single channel
 * and array can be square.
 */
export class Frame {
  readonly labels: ChannelKeys | ChannelNames = [];
  readonly arrays: LazyArray[] = [];

  constructor(
    labelsOrData:
      | FramePayload
      | ChannelParams
      | Map<ChannelKeyOrName, LazyArray[] | LazyArray>
      | Record<ChannelKeyOrName, LazyArray[] | LazyArray> = [],
    arrays: LazyArray | LazyArray[] = []
  ) {
    // Construction from a map.
    if (labelsOrData instanceof Map) {
      labelsOrData.forEach((v, k) => this.push(k, ...toArray(v)));
      return;
    }

    const isObject = typeof labelsOrData === "object" && !Array.isArray(labelsOrData);

    // Construction from a payload.
    if (isObject) {
      if ("keys" in labelsOrData && "arrays" in labelsOrData) {
        const data_ = labelsOrData as FramePayload;
        const arrays = data_.arrays.map((a) => arrayFromPayload(a));
        validateMatchedLabelsAndArrays(data_.keys, arrays);
        data_.keys.forEach((key, i) => this.push(key, arrays[i]));
      } else
        Object.entries(labelsOrData).forEach(([k, v]) => this.push(k, ...toArray(v)));
      return;
    }

    // Construction from a set of arrays and labels.
    if (
      Array.isArray(labelsOrData) ||
      ["string", "number"].includes(typeof labelsOrData)
    ) {
      const data_ = toArray(arrays);
      const labels_ = toArray(labelsOrData) as ChannelKeys | ChannelNames;
      validateMatchedLabelsAndArrays(labels_, data_);
      data_.forEach((d, i) => this.push(labels_[i], d));
      return;
    }

    throw new ValidationError(
      `[Frame] - invalid frame construction parameters. data parameter ust be a frame 
    payload, a list of lazy arrays, a lazy array, a map, or a record keyed by channel 
    name. keys parameter must be a set of channel keys or channel names.`
    );
  }

  /**
   * @returns "key" if the frame is keyed by channel key, "name" if keyed by channel name,
   * and null if the frame is not keyed by channel key or channel name.
   */
  get labeledBy(): "key" | "name" | null {
    if (this.labels.length === 0) return null;
    const firstKey = this.labels[0];
    return typeof firstKey === "string" ? "name" : "key";
  }

  /**
   * @returns the channel keys if the frame is keyed by channel key, and throws an error
   * otherwise.
   */
  get keys(): ChannelKeys {
    if (this.labeledBy === "name") throw new UnexpectedError("keyVariant is not key");
    return (this.labels as ChannelKeys) ?? [];
  }

  /**
   * @returns the unique channel keys if the frame is keyed by channel key, and throws an
   * error otherwise.
   */
  get uniqueKeys(): ChannelKeys {
    return unique(this.keys);
  }

  /**
   * @returns the channel names if the frame is keyed by channel name, and throws an error
   * otherwise.
   */
  get names(): ChannelNames {
    if (this.labeledBy === "key") throw new UnexpectedError("keyVariant is not name");
    return (this.labels as ChannelNames) ?? [];
  }

  /**
   * @returns the unique channel names if the frame is keyed by channel name, and throws an
   * otherwise.
   */
  get uniqueNames(): ChannelNames {
    return unique(this.names);
  }

  /**
   * @returns the unique labels in the frame.
   */
  get uniqueLabels(): ChannelKeys | ChannelNames {
    return this.labeledBy === "key" ? this.uniqueKeys : this.uniqueNames;
  }

  toPayload(): FramePayload {
    return {
      arrays: this.arrays.map((a) => arrayToPayload(a)),
      keys: this.keys,
    };
  }

  /**
   * @returns true if the frame is vertical. Vertical frames are strongly aligned and
   * have on or more channels, but ONLY a single array per channel. Synnax requires
   * that all frames written to the database are vertical.
   */
  get isVertical(): boolean {
    return this.uniqueLabels.length === this.labels.length;
  }

  /**
   * @returns true if the frame is horizontal. Horizontal frames have a single channel,
   * and are strongly aligned by default.A horizontal frame typically has a single array
   * (in which case, it's also 'square'), although it can have multiple arrays if all
   * the arrays are continuous in time.
   */
  get isHorizontal(): boolean {
    return this.uniqueLabels.length === 1;
  }

  /**
   * @returns true if the frame is square. Square frames are both horizontal and vertical.
   * Only a frame with a single channel and array can be square.
   */
  get isSquare(): boolean {
    return this.isHorizontal && this.isVertical;
  }

  /**
   * @returns true if the frame is weakly aligned. A frame is weakly aligned if it meets
   * the time range occupied by all arrays of a particular channel is the same for all
   * channels in the frame. This means that the arrays for a particular channel can have
   * gaps betwen them.
   */
  get isWeaklyAligned(): boolean {
    if (this.labels.length <= 1) return true;
    const ranges = this.timeRanges;
    return ranges.every((tr) => tr.equals(ranges[0]));
  }

  timeRange(label?: ChannelKeyOrName): TimeRange {
    if (label == null) {
      if (this.labels.length === 0) return TimeRange.ZERO;
      const start = Math.min(...this.arrays.map((a) => a.timeRange.start.valueOf()));
      const end = Math.max(...this.arrays.map((a) => a.timeRange.end.valueOf()));
      return new TimeRange(start, end);
    }
    const group = this.get(label);
    if (group == null) return TimeRange.ZERO;
    return new TimeRange(
      group[0].timeRange.start,
      group[group.length - 1].timeRange.end
    );
  }

  get timeRanges(): TimeRange[] {
    return this.uniqueLabels.map((label) => this.timeRange(label));
  }

  /**
   * @returns lazy arrays matching the given channel key or name.
   * @param key the channel key or name.
   */
  get(key: ChannelKeyOrName): LazyArray[];

  /**
   * @returns a frame with the given channel keys or names.
   * @param keys the channel keys or names.
   */
  get(keys: ChannelKeys | ChannelNames): Frame;

  get(key: ChannelKeyOrName | ChannelKeys | ChannelNames): LazyArray[] | Frame {
    if (Array.isArray(key))
      return this.filter((k) => (key as ChannelKeys).includes(k as ChannelKey));
    return this.arrays.filter((_, i) => this.labels[i] === key);
  }

  /**
   * Pushes a set of typed arrays for the given channel onto the frame.
   *
   * @param key the channel key or name;
   * @param v the typed arrays to push.
   */
  push(key: ChannelKeyOrName, ...v: LazyArray[]): void;

  /**
   * Pushes the frame onto the current frame.
   *
   * @param frame  - the frame to push.
   */
  push(frame: Frame): void;

  push(keyOrFrame: ChannelKeyOrName | Frame, ...v: LazyArray[]): void {
    if (keyOrFrame instanceof Frame) {
      if (keyOrFrame.labeledBy !== this.labeledBy)
        throw new ValidationError("keyVariant must match");
      this.arrays.push(...keyOrFrame.arrays);
      (this.labels as ChannelKeys).push(...(keyOrFrame.labels as ChannelKeys));
    } else {
      this.arrays.push(...v);
      if (typeof keyOrFrame === "string" && this.labeledBy === "key")
        throw new ValidationError("keyVariant must match");
      else if (typeof keyOrFrame !== "string" && this.labeledBy === "name")
        throw new ValidationError("keyVariant must match");
      (this.labels as ChannelKeys).push(
        ...(Array.from({ length: v.length }, () => keyOrFrame) as ChannelKeys)
      );
    }
  }

  /**
   * @returns a shallow copy of this frame containing all typed arrays in the current frame and the
   * provided frame.
   */
  concat(frame: Frame): Frame {
    return new Frame([...this.labels, ...frame.labels] as ChannelKeys, [
      ...this.arrays,
      ...frame.arrays,
    ]);
  }

  /**
   * @returns true if the frame contains the provided channel key or name.
   * @param channel the channel key or name to check.
   */
  has(channel: ChannelKeyOrName): boolean {
    if (typeof channel === "string" && this.labeledBy === "key") return false;
    else if (typeof channel === "number" && this.labeledBy === "name") return false;
    return (this.labels as ChannelKeys).includes(channel as ChannelKey);
  }

  /**
   * @returns a new frame containing the mapped output of the provided function.
   * @param fn a function that takes a channel key and typed array and returns a
   * boolean.
   */
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

  /**
   * Iterates over all typed arrays in the current frame.
   *
   * @param fn a function that takes a channel key and typed array.
   */
  forEach(fn: (k: ChannelKeyOrName, arr: LazyArray, i: number) => void): void {
    this.labels.forEach((k, i) => {
      const a = this.arrays[i];
      fn(k, a, i);
    });
  }

  /**
   * @returns a new frame containing all typed arrays in the current frame that pass
   * the provided filter function.
   * @param fn a function that takes a channel key and typed array and returns a boolean.
   */
  filter(fn: (k: ChannelKeyOrName, arr: LazyArray, i: number) => boolean): Frame {
    const frame = new Frame();
    this.labels.forEach((k, i) => {
      const a = this.arrays[i];
      if (fn(k, a, i)) frame.push(k, a);
    });
    return frame;
  }

  /** @returns the total number of bytes in the frame. */
  get byteLength(): Size {
    return new Size(this.arrays.reduce((acc, v) => acc.add(v.byteLength), Size.ZERO));
  }

  /** @returns the total number of samples in the frame. */
  get length(): number {
    return this.arrays.reduce((acc, v) => acc + v.length, 0);
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

const nullTransform = z.null().transform(() => []);

export const frameZ = z.object({
  keys: z.union([nullTransform, z.number().array().optional().default([])]),
  arrays: z.union([nullTransform, array.array().optional().default([])]),
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
