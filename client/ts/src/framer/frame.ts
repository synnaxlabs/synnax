import { ValidationError } from "..";

import { TimeRange, TypedArray } from "@/telem";

export class Frame {
  private readonly keys: string[];
  private readonly arrays: TypedArray[];

  constructor(keys: string[], arrays: TypedArray[]) {
    this.keys = keys;
    this.arrays = arrays;
    this.validate();
  }

  /** @returns true if only one TypedArray exists for each key. */
  get unique(): boolean {
    return this.keys.every((key, i) => this.keys.indexOf(key) === i);
  }

  /** @returns true if the time range and length of each array is equal. */
  get even(): boolean {
    this.validate();
    if (this.arrays.length === 0) return true;
    const first = this.arrays[0];
    return this.arrays.every(
      (a) => a.timeRange.equals(first.timeRange) && a.length === first.length
    );
  }

  /**
   * @returns the widest possible timerange occupied by the frame. If the frame
   */
  get timeRange(): TimeRange {
    this.validate();
    if (this.keys.length === 0) return TimeRange.ZERO;
    // find the minimum start time and maximum end time
    const start = Math.min(...this.arrays.map((a) => a.timeRange.start.valueOf()));
    const end = Math.max(...this.arrays.map((a) => a.timeRange.end.valueOf()));
    return new TimeRange(start, end);
  }

  validate(): void {
    if (this.keys.length !== this.arrays.length)
      throw new ValidationError("keys and arrays must be the same length");
  }

  /**
   * @returns all typed arrays matching the given key. If the frame is unique,
   * this will return an array of length 1.
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
