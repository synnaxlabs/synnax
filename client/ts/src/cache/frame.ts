// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Size, TimeRange, KV } from "@synnaxlabs/x";

import { Frame } from "@/framer";

export interface FrameCacheRequest {
  tr: TimeRange;
  keys: string[];
}

export interface FrameCacheResponse {
  frame: Frame;
  missing: string[];
}

export class FrameCache implements KV<
  FrameCacheRequest,
  FrameCacheResponse,
  TimeRange,
  Frame,
  FrameCacheRequest
> {
  private readonly _cache: Record<string, Frame>;

  constructor() {
    this._cache = {};
  }

  get size(): Size {
    return Object.values(this._cache).reduce((acc, fr) => acc.add(fr.size), Size.ZERO);
  }

  get({ tr, keys }: FrameCacheRequest): FrameCacheResponse {
    const strKey = tr.toString();
    const fr = this._cache[strKey];
    if (fr == null) return { frame: new Frame(), missing: keys };
    const filtered = fr.getF(keys);
    return { frame: filtered, missing: keys.filter((key) => !filtered.has(key)) };
  }

  set(tr: TimeRange, fr: Frame): void {
    const strKey = tr.toString();
    const v = this._cache[strKey];
    if (v == null) this._cache[strKey] = fr;
    else this._cache[strKey] = v.overrideF(fr);
  }

  delete({ tr, keys }: FrameCacheRequest): void {
    const strKey = tr.toString();
    const fr = this._cache[strKey];
    if (fr == null) return;
    this._cache[strKey] = fr.filter((k) => !keys.includes(k));
  }
}

