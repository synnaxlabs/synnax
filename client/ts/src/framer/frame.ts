// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Size, Series, TimeRange, toArray, DataType, unique } from "@synnaxlabs/x";
import { z } from "zod";

import {
  ChannelKey,
  ChannelKeyOrName,
  ChannelKeys,
  ChannelNames,
  ChannelParams,
} from "@/channel/payload";
import { UnexpectedError, ValidationError } from "@/errors";

type ColumnType = "key" | "name" | null;

const columnType = (columns: ChannelParams): ColumnType => {
  const arrKeys = toArray(columns);
  if (arrKeys.length === 0) return null;
  if (typeof arrKeys[0] === "number") return "key";
  return "name";
};

const validateMatchedColsAndArrays = (
  columns: ChannelParams,
  arrays: Series[]
): void => {
  const colsArr = toArray(columns);
  if (colsArr.length === arrays.length) return;
  const colType = columnType(columns);
  if (columnType === null)
    throw new ValidationError(
      "[Frame] - channel keys or names must be provided when constructing a frame."
    );
  throw new ValidationError(
    `[Frame] - ${colType as string}s and arrays must be the same length.
    Got ${colsArr.length} ${colType as string}s and ${arrays.length} arrays.`
  );
};

export type CrudeFrame =
  | Frame
  | FramePayload
  | Map<ChannelKeyOrName, Series[] | Series>
  | Record<ChannelKeyOrName, Series[] | Series>;

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
  readonly columns: ChannelKeys | ChannelNames = [];
  readonly series: Series[] = [];

  constructor(
    columnsOrData: ChannelParams | CrudeFrame = [],
    arrays: Series | Series[] = []
  ) {
    if (columnsOrData instanceof Frame) {
      this.columns = columnsOrData.columns;
      this.series = columnsOrData.series;
      return;
    }

    // Construction from a map.
    if (columnsOrData instanceof Map) {
      columnsOrData.forEach((v, k) => this.push(k, ...toArray(v)));
      return;
    }

    const isObject = typeof columnsOrData === "object" && !Array.isArray(columnsOrData);

    // Construction from a payload.
    if (isObject) {
      if ("keys" in columnsOrData && "series" in columnsOrData) {
        const data_ = columnsOrData as FramePayload;
        const arrays = data_.series.map((a) => arrayFromPayload(a));
        validateMatchedColsAndArrays(data_.keys, arrays);
        data_.keys.forEach((key, i) => this.push(key, arrays[i]));
      } else
        Object.entries(columnsOrData).forEach(([k, v]) => this.push(k, ...toArray(v)));
      return;
    }

    // Construction from a set of arrays and columns.
    if (
      Array.isArray(columnsOrData) ||
      ["string", "number"].includes(typeof columnsOrData)
    ) {
      const data_ = toArray(arrays);
      const cols = toArray(columnsOrData) as ChannelKeys | ChannelNames;
      validateMatchedColsAndArrays(cols, data_);
      data_.forEach((d, i) => this.push(cols[i], d));
      return;
    }

    throw new ValidationError(
      `[Frame] - invalid frame construction parameters. data parameter ust be a frame 
    payload, a list of lazy arrays, a lazy array, a map, or a record keyed by channel 
    name. keys parameter must be a set of channel keys or channel names.`
    );
  }

  /**
   * @returns "key" if the frame columns are channel keys, "name" if the columns are
   * channel names, and null if the frame has no columns.
   */
  get colType(): ColumnType {
    if (this.columns.length === 0) return null;
    const firstKey = this.columns[0];
    return typeof firstKey === "string" ? "name" : "key";
  }

  /**
   * @returns the channel keys if the frame is keyed by channel key, and throws an error
   * otherwise.
   */
  get keys(): ChannelKeys {
    console.log(this.columns);
    if (this.colType === "name") throw new UnexpectedError("colType is not key");
    return (this.columns as ChannelKeys) ?? [];
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
    if (this.colType === "key") throw new UnexpectedError("colType is not name");
    return (this.columns as ChannelNames) ?? [];
  }

  /**
   * @returns the unique channel names if the frame is keyed by channel name, and throws an
   * otherwise.
   */
  get uniqueNames(): ChannelNames {
    return unique(this.names);
  }

  /**
   * @returns the unique columns in the frame.
   */
  get uniqueColumns(): ChannelKeys | ChannelNames {
    return this.colType === "key" ? this.uniqueKeys : this.uniqueNames;
  }

  toPayload(): FramePayload {
    return {
      series: this.series.map((a) => arrayToPayload(a)),
      keys: this.keys,
    };
  }

  /**
   * @returns true if the frame is vertical. Vertical frames are strongly aligned and
   * have on or more channels, but ONLY a single array per channel. Synnax requires
   * that all frames written to the database are vertical.
   */
  get isVertical(): boolean {
    return this.uniqueColumns.length === this.columns.length;
  }

  /**
   * @returns true if the frame is horizontal. Horizontal frames have a single channel,
   * and are strongly aligned by default.A horizontal frame typically has a single array
   * (in which case, it's also 'square'), although it can have multiple arrays if all
   * the arrays are continuous in time.
   */
  get isHorizontal(): boolean {
    return this.uniqueColumns.length === 1;
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
    if (this.columns.length <= 1) return true;
    const ranges = this.timeRanges;
    return ranges.every((tr) => tr.equals(ranges[0]));
  }

  timeRange(col?: ChannelKeyOrName): TimeRange {
    if (col == null) {
      if (this.columns.length === 0) return TimeRange.ZERO;
      const start = Math.min(...this.series.map((a) => a.timeRange.start.valueOf()));
      const end = Math.max(...this.series.map((a) => a.timeRange.end.valueOf()));
      return new TimeRange(start, end);
    }
    const group = this.get(col);
    if (group == null) return TimeRange.ZERO;
    return new TimeRange(
      group[0].timeRange.start,
      group[group.length - 1].timeRange.end
    );
  }

  get timeRanges(): TimeRange[] {
    return this.uniqueColumns.map((col) => this.timeRange(col));
  }

  /**
   * @returns lazy arrays matching the given channel key or name.
   * @param key the channel key or name.
   */
  get(key: ChannelKeyOrName): Series[];

  /**
   * @returns a frame with the given channel keys or names.
   * @param keys the channel keys or names.
   */
  get(keys: ChannelKeys | ChannelNames): Frame;

  get(key: ChannelKeyOrName | ChannelKeys | ChannelNames): Series[] | Frame {
    if (Array.isArray(key))
      return this.filter((k) => (key as ChannelKeys).includes(k as ChannelKey));
    return this.series.filter((_, i) => this.columns[i] === key);
  }

  /**
   * Pushes a set of typed arrays for the given channel onto the frame.
   *
   * @param key the channel key or name;
   * @param v the typed arrays to push.
   */
  push(key: ChannelKeyOrName, ...v: Series[]): void;

  /**
   * Pushes the frame onto the current frame.
   *
   * @param frame  - the frame to push.
   */
  push(frame: Frame): void;

  push(keyOrFrame: ChannelKeyOrName | Frame, ...v: Series[]): void {
    if (keyOrFrame instanceof Frame) {
      if (this.colType !== null && keyOrFrame.colType !== this.colType)
        throw new ValidationError("keyVariant must match");
      this.series.push(...keyOrFrame.series);
      (this.columns as ChannelKeys).push(...(keyOrFrame.columns as ChannelKeys));
    } else {
      this.series.push(...v);
      if (typeof keyOrFrame === "string" && this.colType === "key")
        throw new ValidationError("keyVariant must match");
      else if (typeof keyOrFrame !== "string" && this.colType === "name")
        throw new ValidationError("keyVariant must match");
      (this.columns as ChannelKeys).push(
        ...(Array.from({ length: v.length }, () => keyOrFrame) as ChannelKeys)
      );
    }
  }

  /**
   * @returns a shallow copy of this frame containing all typed arrays in the current frame and the
   * provided frame.
   */
  concat(frame: Frame): Frame {
    return new Frame([...this.columns, ...frame.columns] as ChannelKeys, [
      ...this.series,
      ...frame.series,
    ]);
  }

  /**
   * @returns true if the frame contains the provided channel key or name.
   * @param channel the channel key or name to check.
   */
  has(channel: ChannelKeyOrName): boolean {
    if (typeof channel === "string" && this.colType === "key") return false;
    else if (typeof channel === "number" && this.colType === "name") return false;
    return (this.columns as ChannelKeys).includes(channel as ChannelKey);
  }

  /**
   * @returns a new frame containing the mapped output of the provided function.
   * @param fn a function that takes a channel key and typed array and returns a
   * boolean.
   */
  map(
    fn: (k: ChannelKeyOrName, arr: Series, i: number) => [ChannelKeyOrName, Series]
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
  forEach(fn: (k: ChannelKeyOrName, arr: Series, i: number) => void): void {
    this.columns.forEach((k, i) => {
      const a = this.series[i];
      fn(k, a, i);
    });
  }

  /**
   * @returns a new frame containing all typed arrays in the current frame that pass
   * the provided filter function.
   * @param fn a function that takes a channel key and typed array and returns a boolean.
   */
  filter(fn: (k: ChannelKeyOrName, arr: Series, i: number) => boolean): Frame {
    const frame = new Frame();
    this.columns.forEach((k, i) => {
      const a = this.series[i];
      if (fn(k, a, i)) frame.push(k, a);
    });
    return frame;
  }

  /** @returns the total number of bytes in the frame. */
  get byteLength(): Size {
    return new Size(this.series.reduce((acc, v) => acc.add(v.byteLength), Size.ZERO));
  }

  /** @returns the total number of samples in the frame. */
  get length(): number {
    return this.series.reduce((acc, v) => acc + v.length, 0);
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
  keys: z.union([
    z.null().transform(() => [] as number[]),
    z.number().array().optional().default([]),
  ]),
  series: z.union([
    z.null().transform(() => [] as Array<z.infer<typeof array>>),
    array.array().optional().default([]),
  ]),
});

export type FramePayload = z.infer<typeof frameZ>;

export const arrayFromPayload = (payload: ArrayPayload): Series => {
  const { dataType, data, timeRange } = payload;
  return new Series(data, dataType, timeRange);
};

export const arrayToPayload = (array: Series): ArrayPayload => {
  return {
    timeRange: array._timeRange,
    dataType: array.dataType,
    data: new Uint8Array(array.data.buffer),
  };
};
