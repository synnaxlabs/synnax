// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  MultiSeries,
  Series,
  type SeriesDigest,
  type SeriesPayload,
  Size,
  type TelemValue,
  TimeRange,
  TimeStamp,
  unique,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type channel } from "@/channel";
import { UnexpectedError, ValidationError } from "@/errors";

type ColumnType = "key" | "name" | null;

export interface Digest extends Record<channel.KeyOrName, SeriesDigest[]> {}

const columnType = (columns: channel.PrimitiveParams): ColumnType => {
  const arrKeys = array.toArray(columns);
  if (arrKeys.length === 0) return null;
  if (typeof arrKeys[0] === "number") return "key";
  if (!isNaN(parseInt(arrKeys[0]))) return "key";
  return "name";
};

const validateMatchedColsAndSeries = (
  columns: channel.PrimitiveParams,
  series: Series[],
): void => {
  const colsArr = array.toArray(columns);
  if (colsArr.length === series.length) return;
  const colType = columnType(columns);
  if (columnType === null)
    throw new ValidationError(
      "[Frame] - channel keys or names must be provided when constructing a frame.",
    );
  throw new ValidationError(
    `[Frame] - ${colType as string}s and series must be the same length.
    Got ${colsArr.length} ${colType as string}s and ${series.length} series.`,
  );
};

export type CrudeFrame =
  | Frame
  | CrudePayload
  | Map<channel.KeyOrName, Series[] | Series>
  | Record<channel.KeyOrName, Series[] | Series>;

/**
 * A frame is a collection of series mapped to a particular channel. Frames
 * can be keyed by channel name or channel key, but not both.
 *
 * Frames have two important characteristics: alignment and orientation.
 *
 * A frame's alignment defines how correlated the series for different channels in the
 * frame are:
 *
 * - A frame is weakly aligned if the time range occupied by all series of a
 * particular channel is the same for all channels in the frame. This means that the
 * series for a particular channel can have gaps between them.
 *
 * - A strongly aligned frame means that all channels share the same rate/index and
 * there are no gaps in time between series. Strongly aligned frames are natural
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
 * although it can have multiple series if all the series are continuous in time.
 *
 * - Vertical frames are strongly aligned and have on or more channels, but ONLY a single
 * array per channel. Synnax requires that all frames written to the database are
 * vertical.
 *
 * - Square frames are both horizontal and vertical. Only a frame with a single channel
 * and array can be square.
 */
export class Frame {
  readonly columns: channel.Keys | channel.Names = [];
  readonly series: Series[] = [];

  constructor(
    columnsOrData: channel.PrimitiveParams | CrudeFrame = [],
    series: Series | Series[] = [],
  ) {
    if (columnsOrData instanceof Frame) {
      this.columns = columnsOrData.columns;
      this.series = columnsOrData.series;
      return;
    }

    // Construction from a map.
    if (columnsOrData instanceof Map) {
      columnsOrData.forEach((v, k) => this.push(k, ...array.toArray(v)));
      return;
    }

    const isObject = typeof columnsOrData === "object" && !Array.isArray(columnsOrData);

    // Construction from a payload.
    if (isObject) {
      if ("keys" in columnsOrData && "series" in columnsOrData) {
        const data_ = columnsOrData as Payload;
        data_.series ??= [];
        data_.keys ??= [];
        const series = data_.series.map((a) => seriesFromPayload(a));
        validateMatchedColsAndSeries(data_.keys, series);
        data_.keys.forEach((key, i) => this.push(key, series[i]));
      } else
        Object.entries(columnsOrData).forEach(([k, v]) => {
          const key = parseInt(k);
          if (!isNaN(key)) return this.push(key, ...array.toArray(v));
          this.push(k, ...array.toArray(v));
        });
      return;
    }

    // Construction from a set of series and columns.
    if (
      Array.isArray(columnsOrData) ||
      ["string", "number"].includes(typeof columnsOrData)
    ) {
      const data_ = array.toArray(series);
      const cols = array.toArray(columnsOrData) as channel.Keys | channel.Names;
      validateMatchedColsAndSeries(cols, data_);
      data_.forEach((d, i) => this.push(cols[i], d));
      return;
    }

    throw new ValidationError(
      `[Frame] - invalid frame construction parameters. data parameter ust be a frame
    payload, a list of lazy series, a lazy array, a map, or a record keyed by channel
    name. keys parameter must be a set of channel keys or channel names.`,
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
  get keys(): channel.Keys {
    if (this.colType === "name") throw new UnexpectedError("colType is not key");
    return (this.columns as channel.Keys) ?? [];
  }

  /**
   * @returns the unique channel keys if the frame is keyed by channel key, and throws an
   * error otherwise.
   */
  get uniqueKeys(): channel.Keys {
    return unique.unique(this.keys);
  }

  /**
   * @returns the channel names if the frame is keyed by channel name, and throws an error
   * otherwise.
   */
  get names(): channel.Names {
    if (this.colType === "key") throw new UnexpectedError("colType is not name");
    return (this.columns as channel.Names) ?? [];
  }

  /**
   * @returns the unique channel names if the frame is keyed by channel name, and throws an
   * otherwise.
   */
  get uniqueNames(): channel.Names {
    return unique.unique(this.names);
  }

  /**
   * @returns the unique columns in the frame.
   */
  get uniqueColumns(): channel.Keys | channel.Names {
    return this.colType === "key" ? this.uniqueKeys : this.uniqueNames;
  }

  toPayload(): Payload {
    return {
      series: this.series.map((a) => seriesToPayload(a)),
      keys: [...this.keys],
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
   * and are strongly aligned by default. A horizontal frame typically has a single
   * array (in which case, it's also 'square'), although it can have multiple series if
   * all the series are continuous in time.
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
   * the time range occupied by all series of a particular channel is the same for all
   * channels in the frame. This means that the series for a particular channel can have
   * gaps between them.
   */
  get isWeaklyAligned(): boolean {
    if (this.columns.length <= 1) return true;
    const ranges = this.timeRanges;
    return ranges.every((tr) => tr.equals(ranges[0]));
  }

  timeRange(col?: channel.KeyOrName): TimeRange {
    if (col == null) {
      if (this.columns.length === 0) return TimeRange.ZERO;
      const start = TimeStamp.min(...this.series.map((a) => a.timeRange.start));
      const end = TimeStamp.max(...this.series.map((a) => a.timeRange.end));
      return new TimeRange(start, end);
    }
    const group = this.get(col);
    if (group == null) return TimeRange.ZERO;
    return group.timeRange;
  }

  latest(): Record<string, TelemValue | undefined> {
    return this.at(-1);
  }

  get timeRanges(): TimeRange[] {
    return this.uniqueColumns.map((col) => this.timeRange(col));
  }

  /**
   * @returns lazy series matching the given channel key or name.
   * @param key the channel key or name.
   */
  get(key: channel.KeyOrName): MultiSeries;

  /**
   * @returns a frame with the given channel keys or names.
   * @param keys the channel keys or names.
   */
  get(keys: channel.Keys | channel.Names): Frame;

  get(key: channel.KeyOrName | channel.Keys | channel.Names): MultiSeries | Frame {
    if (Array.isArray(key))
      return this.filter((k) => (key as channel.Keys).includes(k as channel.Key));
    return new MultiSeries(this.series.filter((_, i) => this.columns[i] === key));
  }

  /**
   * Pushes a set of series for the given channel onto the frame.
   *
   * @param key the channel key or name;
   * @param v the series to push.
   */
  push(key: channel.KeyOrName, ...v: Series[]): void;

  /**
   * Pushes the frame onto the current frame.
   *
   * @param frame  - the frame to push.
   */
  push(frame: Frame): void;

  push(keyOrFrame: channel.KeyOrName | Frame, ...v: Series[]): void {
    if (keyOrFrame instanceof Frame) {
      if (
        keyOrFrame.colType != null &&
        this.colType !== null &&
        keyOrFrame.colType !== this.colType
      )
        throw new ValidationError("keyVariant must match");
      this.series.push(...keyOrFrame.series);
      (this.columns as channel.Keys).push(...(keyOrFrame.columns as channel.Keys));
    } else {
      this.series.push(...v);
      if (typeof keyOrFrame === "string" && this.colType === "key")
        throw new ValidationError("keyVariant must match");
      else if (typeof keyOrFrame !== "string" && this.colType === "name")
        throw new ValidationError("keyVariant must match");
      (this.columns as channel.Keys).push(
        ...(Array.from({ length: v.length }, () => keyOrFrame) as channel.Keys),
      );
    }
  }

  /**
   * @returns a shallow copy of this frame containing all series in the current frame and the
   * provided frame.
   */
  concat(frame: Frame): Frame {
    return new Frame([...this.columns, ...frame.columns] as channel.Keys, [
      ...this.series,
      ...frame.series,
    ]);
  }

  /**
   * @returns true if the frame contains the provided channel key or name.
   * @param channel the channel key or name to check.
   */
  has(channel: channel.KeyOrName): boolean {
    if (typeof channel === "string" && this.colType === "key") return false;
    if (typeof channel === "number" && this.colType === "name") return false;
    return (this.columns as channel.Keys).includes(channel as channel.Key);
  }

  /**
   * @returns a new frame containing the mapped output of the provided function.
   * @param fn a function that takes a channel key and series and returns a
   * boolean.
   */
  map(
    fn: (k: channel.KeyOrName, arr: Series, i: number) => [channel.KeyOrName, Series],
  ): Frame {
    const frame = new Frame();
    this.forEach((k, arr, i) => frame.push(...fn(k, arr, i)));
    return frame;
  }

  mapFilter(
    fn: (
      k: channel.KeyOrName,
      arr: Series,
      i: number,
    ) => [channel.KeyOrName, Series, boolean],
  ): Frame {
    const frame = new Frame();
    this.forEach((k, arr, i) => {
      const [newK, newArr, keep] = fn(k, arr, i);
      if (keep) frame.push(newK, newArr);
    });
    return frame;
  }

  /**
   * Iterates over all series in the current frame.
   *
   * @param fn a function that takes a channel key and series.
   */
  forEach(fn: (k: channel.KeyOrName, arr: Series, i: number) => void): void {
    this.columns.forEach((k, i) => {
      const a = this.series[i];
      fn(k, a, i);
    });
  }

  at(index: number, required: true): Record<channel.KeyOrName, TelemValue>;

  at(
    index: number,
    required?: false,
  ): Record<channel.KeyOrName, TelemValue | undefined>;

  at(
    index: number,
    required = false,
  ): Record<channel.KeyOrName, TelemValue | undefined> {
    const res: Record<channel.KeyOrName, TelemValue> = {};
    this.uniqueColumns.forEach((k) => {
      res[k] = this.get(k).at(index, required as true);
    });
    return res;
  }

  /**
   * @returns a new frame containing all series in the current frame that pass
   * the provided filter function.
   * @param fn a function that takes a channel key and series and returns a boolean.
   */
  filter(fn: (k: channel.KeyOrName, arr: Series, i: number) => boolean): Frame {
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

  /**
   * @returns a digest of information about the structure of the frame for debugging
   * purposes.
   */
  get digest(): Digest {
    const digest: Digest = {};
    this.keys.forEach((k, i) => {
      const sd = this.series[i].digest;
      if (k in digest) digest[k].push(sd);
      else digest[k] = [sd];
    });
    return digest;
  }

  /** @returns the total number of samples in the frame. */
  get length(): number {
    return this.series.reduce((acc, v) => acc + v.length, 0);
  }

  toString(): string {
    let str = `Frame{\n`;
    this.uniqueColumns.forEach((c) => {
      str += `  ${c}: ${this.get(c)
        .series.map((s) => s.toString())
        .join(",")}\n`;
    });
    str += "}";
    return str;
  }
}

export const frameZ = z.object({
  keys: z.union([
    z.null().transform<number[]>(() => []),
    z.number().array().optional().default([]),
  ]),
  series: z.union([
    z.null().transform<z.infer<typeof Series.crudeZ>[]>(() => []),
    Series.crudeZ.array().optional().default([]),
  ]),
});

export interface Payload extends z.infer<typeof frameZ> {}

export interface CrudePayload extends z.input<typeof frameZ> {}

export const seriesFromPayload = (series: SeriesPayload): Series => {
  const { dataType, data, timeRange, alignment } = series;
  return new Series({ data, dataType, timeRange, glBufferUsage: "static", alignment });
};

export const seriesToPayload = (series: Series): SeriesPayload => ({
  timeRange: series.timeRange,
  dataType: series.dataType,
  data: new Uint8Array(series.data.buffer),
  alignment: series.alignment,
});
