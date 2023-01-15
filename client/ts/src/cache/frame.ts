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

import { Frame } from "@/framer";
import { Size, TimeRange } from "@synnaxlabs/x";

export class FrameCache {
  private readonly _cache: Record<string, Frame>;

  constructor() {
    this._cache = {};
  }

  get size(): Size {
    return Object.values(this._cache).reduce((acc, fr) => acc.add(fr.size), Size.ZERO);
  }

  get(tr: TimeRange, ...keys: string[]): FrameCacheResult {
    const strKey = tr.toString();
    const fr = this._cache[strKey];
    if (fr == null) return { frame: new Frame(), missing: keys };
    const filtered = fr.getF(keys);
    return { frame: filtered, missing: keys.filter((key) => !filtered.has(key)) };
  }

  overrideF(tr: TimeRange, fr: Frame): void {
    const v = this._cache[this.key(tr)];
    if (v == null) this._cache[this.key(tr)] = fr;
    else this._cache[this.key(tr)] = v.overrideF(fr);
  }

  private key(tr: TimeRange): string {
    return tr.toString();
  }
}

export interface FrameCacheResult {
  frame: Frame;
  missing: string[];
}
