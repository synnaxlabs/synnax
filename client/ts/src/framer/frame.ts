import { TelemArray, UnexpectedError, ValidationError } from "..";

import { arrayFromPayload, arrayToPayload, FramePayload } from "./payload";

import { Size, TArray, TimeRange } from "@/telem";

export class Frame {
  private readonly _entries: Record<string, TArray[]>;

  constructor(
    data: FramePayload | Record<string, TArray[]> | TArray[] | TArray = [],
    keys: string | string[] = []
  ) {
    this._entries = {};
    if (Array.isArray(data)) {
      if (keys.length !== data.length)
        throw new ValidationError("keys and data must be the same length");
      data.forEach((d, i) => this.pushA(keys[i], d));
    } else if ("keys" in data) {
      const v = data as FramePayload;
      if (v.arrays == null || v.keys == null || v.keys.length !== v.arrays.length)
        throw new ValidationError("arrays and keys must be defined");
      v.keys.forEach((key, i) =>
        this.pushA(key, arrayFromPayload((v.arrays as TelemArray[])[i]))
      );
    } else if (data instanceof TArray) {
      this.pushA(keys as string, data);
    } else {
      this._entries = data;
    }
  }

  toPayload(): FramePayload {
    return {
      arrays: this.arrays.map((a) => arrayToPayload(a)),
      keys: this.keys,
    };
  }

  get isVertical(): boolean {
    return Object.values(this._entries).every((v) => v.length === 1);
  }

  get isHorizontal(): boolean {
    return Object.keys(this._entries).length === 1;
  }

  get isWeaklyAligned(): boolean {
    if (this.keys.length <= 1) return true;
    return this.timeRanges.every((tr) => tr.equals(this.timeRanges[0]));
  }

  timeRange(key?: string): TimeRange {
    if (key == null) {
      if (this.keys.length === 0) return TimeRange.ZERO;
      const start = Math.min(...this.arrays.map((a) => a.timeRange.start.valueOf()));
      const end = Math.max(...this.arrays.map((a) => a.timeRange.end.valueOf()));
      return new TimeRange(start, end);
    }
    const group = this.getA(key);
    if (group == null) return TimeRange.ZERO;
    return new TimeRange(
      group[0].timeRange.start,
      group[group.length - 1].timeRange.end
    );
  }

  get timeRanges(): TimeRange[] {
    return this.keys.map((key) => this.timeRange(key));
  }

  /**
   * @returns all typed arrays matching the given key. If the frame is vertical,
   * this will return an array of length 1. If the frame is horiztonal, returns all
   * arrays in the frame.
   */
  getA(key: string): TArray[] {
    return this._entries[key] ?? [];
  }

  /**
   * @returns a new frame containing only the typed arrays matching the given keys.
   * @param keys - the keys to filter by.
   */
  getF(keys: string[]): Frame {
    const frame = new Frame();
    for (const key of keys) {
      const e = this._entries[key];
      if (e == null) continue;
      frame._entries[key] = e;
    }
    return frame;
  }

  /**
   *
   * @param key - the key to filter by.
   */
  pushA(key: string, ...v: TArray[]): void {
    this._entries[key] = (this._entries[key] ?? []).concat(v);
  }

  overrideA(key: string, ...v: TArray[]): void {
    this._entries[key] = v;
  }

  pushF(frame: Frame): Frame {
    for (const [key, arrays] of frame.entries) {
      this._entries[key] = (this._entries[key] ?? []).concat(arrays);
    }
    return this;
  }

  overrideF(frame: Frame): Frame {
    for (const [key, arrays] of frame.entries) {
      this._entries[key] = arrays;
    }
    return this;
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

  get keys(): string[] {
    return Object.keys(this._entries);
  }

  get entries(): Array<[string, TArray[]]> {
    return Object.entries(this._entries);
  }

  get arrays(): TArray[] {
    return Object.values(this._entries).flat();
  }

  map(fn: (k: string, arr: TArray, i: number) => TArray): Frame {
    const frame = new Frame();
    for (const [k, a] of this.entries) {
      frame._entries[k] = a.map((arr, i) => fn(k, arr, i));
    }
    return frame;
  }

  filter(fn: (k: string, arr: TArray, i: number) => boolean): Frame {
    const f = new Frame();
    for (const [k, a] of this.entries) {
      if (a == null) throw new UnexpectedError(`a is null for key ${k}`);
      const filtered = a.filter((arr, i) => fn(k, arr, i));
      if (filtered.length > 0) f._entries[k] = filtered;
    }
    return f;
  }

  get size(): Size {
    return new Size(this.arrays.reduce((acc, v) => acc.add(v.size), Size.ZERO));
  }
}
