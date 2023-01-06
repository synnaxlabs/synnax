import { arrayFromPayload, FramePayload } from "./payload";

import { TimeRange, TelemArray, Size } from "@/telem";

export class Frame {
  private readonly _entries: Record<string, TelemArray[]>;

  constructor() {
    this._entries = {};
  }

  static fromPayload(fp: FramePayload): Frame {
    const f = new Frame();
    fp.keys.forEach((key, i) => f.pushA(key, arrayFromPayload(fp.arrays[i])));
    return f;
  }

  toPayload(): FramePayload {
    return {
      arrays: this.arrays,
      keys: this.keys,
    };
  }

  get vertical(): boolean {
    return Object.values(this._entries).every((v) => v.length === 1);
  }

  get horizontal(): boolean {
    return Object.keys(this._entries).length === 1;
  }

  get weaklyAligned(): boolean {
    if (this.keys.length <= 1) return true;
    const timeRanges = this.timeRanges();
    return timeRanges.every((tr) => tr.equals(timeRanges[0]));
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

  timeRanges(): TimeRange[] {
    return this.keys.map((key) => this.timeRange(key));
  }

  /**
   * @returns all typed arrays matching the given key. If the frame is vertical,
   * this will return an array of length 1. If the frame is horiztonal, returns all
   * arrays in the frame.
   */
  getA(key: string): TelemArray[] {
    return this._entries[key] ?? [];
  }

  /**
   * @returns a new frame containing only the typed arrays matching the given keys.
   * @param keys - the keys to filter by.
   */
  getF(keys: string[]): Frame {
    const frame = new Frame();
    for (const key of keys) {
      frame._entries[key] = this._entries[key];
    }
    return frame;
  }

  /**
   *
   * @param key - the key to filter by.
   */
  pushA(key: string, ...v: TelemArray[]): void {
    this._entries[key] = (this._entries[key] ?? []).concat(v);
  }

  overrideA(key: string, ...v: TelemArray[]): void {
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

  get entries(): Array<[string, TelemArray[]]> {
    return Object.entries(this._entries);
  }

  get arrays(): TelemArray[] {
    return Object.values(this._entries).flat();
  }

  map(fn: (v: TelemArray, k: string, i: number) => TelemArray): Frame {
    const frame = new Frame();
    for (const [key, arrays] of this.entries) {
      frame._entries[key] = arrays.map((v, i) => fn(v, key, i));
    }
    return frame;
  }

  filter(fn: (v: TelemArray, k: string, i: number) => boolean): Frame {
    const f = new Frame();
    for (const [k, a] of this.entries) f._entries[k] = a.filter((v, i) => fn(v, k, i));
    return f;
  }

  size(): Size {
    return new Size(this.arrays.reduce((acc, v) => acc.add(v.size), Size.ZERO));
  }
}
