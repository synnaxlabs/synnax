// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export class GLCache {
  gl: WebGLRenderingContext;
  entries: Record<string, Record<string, CacheEntry>>;

  constructor(gl: WebGLRenderingContext) {
    this.gl = gl;
    this.entries = {};
  }

  get(rangeKey: string, channelKey: string): WebGLBuffer[] | undefined {
    const range = this.entries[rangeKey];
    if (range == null) return undefined;
    return range[channelKey].buffers;
  }

  buffer(rangeKey: string, channelKey: string);
}

class CacheEntry {
  buffers: WebGLBuffer[];

  constructor(buffers: WebGLBuffer[]) {
    this.buffers = buffers;
  }
}
