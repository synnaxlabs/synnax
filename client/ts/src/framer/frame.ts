/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

import { Size, LazyArray, TimeRange } from "@synnaxlabs/x";

import { arrayFromPayload, arrayToPayload, FramePayload } from "./payload";

import { UnexpectedError, ValidationError } from "@/errors";

export class Frame {
  private readonly _entries: Record<string, LazyArray[]>;

  constructor(
    data: FramePayload | Record<string, LazyArray[]> | LazyArray[] | LazyArray = [],
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
        this.pushA(key, arrayFromPayload((v.arrays as LazyArray[])[i]))
      );
    } else if (data instanceof LazyArray) {
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
  getA(key: string): LazyArray[] {
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
  pushA(key: string, ...v: LazyArray[]): void {
    this._entries[key] = (this._entries[key] ?? []).concat(v);
  }

  /**
  * @returns a shallow copy of this frame with the given key overridden with the
  * provided typed arrays.
  */
  overrideA(key: string, ...v: LazyArray[]): Frame {
    const next = this.shallowCopy();
    next._entries[key] = v;
    return next;
  }

  /**
  * @returns a shallow copy of this frame containing all typed arrays in the current frame and the
  * provided frame.
  */
  concatF(frame: Frame): Frame {
    const next = this.shallowCopy();
    for (const [key, arrays] of frame.entries) {
      next._entries[key] = (next._entries[key] ?? []).concat(arrays);
    }
    return next;
  }

  /**
  * @returns a shallow copy of the frame with the provided frame's entries
  * overriding the current frame's entries i.e. all typed arrays in the
  * provided frame will replace the current frame's typed arrays with the
  * same key.
  */
  overrideF(frame: Frame): Frame {
    const next = this.shallowCopy();
    for (const [key, arrays] of frame.entries) {
      next._entries[key] = arrays;
    }
    return next;
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

  get entries(): Array<[string, LazyArray[]]> {
    return Object.entries(this._entries);
  }

  get arrays(): LazyArray[] {
    return Object.values(this._entries).flat();
  }

  map(fn: (k: string, arr: LazyArray, i: number) => LazyArray): Frame {
    const frame = new Frame();
    for (const [k, a] of this.entries) {
      frame._entries[k] = a.map((arr, i) => fn(k, arr, i));
    }
    return frame;
  }

  filter(fn: (k: string, arr: LazyArray, i: number) => boolean): Frame {
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

  shallowCopy(): Frame {
    const fr = new Frame()
    for (const [k, a] of this.entries) {
      fr._entries[k] = a.slice();
    }
    return fr;
  }
}
