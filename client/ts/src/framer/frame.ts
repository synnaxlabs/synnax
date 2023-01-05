import { ValidationError } from "..";

import { TimeRange, TypedArray, Frame as CoreFrame } from "@/telem";
import { unique } from "@/util/unique";

export class Frame extends CoreFrame {
  private readonly keys: string[];

  constructor(keys: string[], arrays: TypedArray[] = []) {
    super(arrays);
    this.keys = keys;
    this.validate();
  }

  /** @returns true if only one TypedArray exists for each key. */
  get vertical(): boolean {
    return unique(this.keys).length === this.arrays.length;
  }

  get horizontal(): boolean {
    return unique(this.keys).length === 1;
  }

  /** @returns true if the frame is square i.e. the minimum and maximum timestamps of each
   * key are the same. */
  get square(): boolean {
    this.validate();
    if (this.arrays.length <= 1) return true;

    const groups = this.groupByKey();
    const first = groups[0];
    const base = new TimeRange(
      first[0].timeRange.start,
      first[first.length - 1].timeRange.end
    );

    return Object.values(groups).every((group) =>
      new TimeRange(
        group[0].timeRange.start,
        group[group.length - 1].timeRange.end
      ).equals(base)
    );
  }

  /**
   * @returns the widest possible timerange occupied by the frame.
   */
  get timeRange(): TimeRange {
    this.validate();
    if (this.keys.length === 0) return TimeRange.ZERO;
    // find the minimum start time and maximum end time
    const start = Math.min(...this.arrays.map((a) => a.timeRange.start.valueOf()));
    const end = Math.max(...this.arrays.map((a) => a.timeRange.end.valueOf()));
    return new TimeRange(start, end);
  }

  private groupByKey(): Record<string, TypedArray[]> {
    return this.keys.reduce<Record<string, TypedArray[]>>((acc, key, i) => {
      const curr = acc[key];
      if (curr == null) acc[key] = [this.arrays[i]];
      curr.push(this.arrays[i]);
      curr.sort((a, b) => a.timeRange.start.valueOf() - b.timeRange.start.valueOf());
      return acc;
    }, {});
  }

  validate(): void {
    if (this.keys.length !== this.arrays.length)
      throw new ValidationError("keys and arrays must be the same length");
  }

  /**
   * @returns all typed arrays matching the given key. If the frame is vertical,
   * this will return an array of length 1. If the frame is horiztonal, returns all
   * arrays in the frame.
   */
  get(key: string): TypedArray[] {
    return this.arrays.filter((_, i) => this.keys[i] === key);
  }

  add(key: string, v: TypedArray): void {
    this.keys.push(key);
    this.arrays.push(v);
  }

  /**
   * Checks whether the frame has a typed array matching the given key.
   *
   * @param key - the key to check for.
   * @returns true if the frame has a typed array matching the given key.
   */
  has(key: string): boolean {
    return this.keys.includes(key);
  }
}
