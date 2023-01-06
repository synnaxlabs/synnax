// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Frame, ValidationError } from "@synnaxlabs/client";

import { unique } from "@/util/unique";

export class FrameCache {
  entries: Record<string, Frame>;

  constructor() {
    this.entries = {};
  }

  get(req: FrameCacheGetRequest): FrameGetGetResponse {
    this.validateRequest(req);

    const { range, keys } = req;
    const frame = this.entries[range];
    if (frame == null)
      return {
        frame: new Frame(),
        missing: keys,
      };

    const values = Object.fromEntries(
      keys.map((key) => [key, frame.get(key)]).filter(([, value]) => value.length > 0)
    );
    const missing = keys.filter((key) => !(key in values));

    return {
      frame: new Frame(Object.keys(values), Object.values(values)),
      missing,
    };
  }

  set(req: FrameCacheSetRequest): void {
    const existing = this.entries[req.range];
    if (existing == null) {
      this.entries[req.range] = req.frame;
      return;
    }
    req.frame.forEach(([key, value]) => {
      existing.delete(key);
      existing.add(key, value);
    });
    this.entries[req.range] = existing;
  }

  private validateRequest(req: FrameCacheGetRequest): void {
    if (unique(req.keys).length !== req.keys.length)
      throw new ValidationError("Duplicate keys in request");
  }
}

export interface FrameCacheGetRequest {
  range: string;
  keys: string[];
}

export interface FrameGetGetResponse {
  frame: Frame | null;
  missing: string[];
}

export interface FrameCacheSetRequest {
  range: string;
  frame: Frame;
}
