// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TelemArray, UnexpectedError } from "@synnaxlabs/client";

export interface WebGLBufferController {
  createBuffer: () => WebGLBuffer | null;
  bindBuffer: (target: number, buffer: WebGLBuffer | null) => void;
  bufferData: (target: number, size: ArrayBufferLike, usage: number) => void;
  deleteBuffer: (buffer: WebGLBuffer) => void;
  ARRAY_BUFFER: number;
  STATIC_DRAW: number;
}

export class WebGLBufferCache {
  gl: WebGLBufferController;
  entries: Record<string, Record<string, CacheEntry>>;

  constructor(gl: WebGLBufferController) {
    this.gl = gl;
    this.entries = {};
  }

  get(range: string, channel: string): WebGLBuffer[] | undefined {
    const rng = this.entries[range];
    if (range == null) return undefined;
    const entry = rng[channel];
    if (entry == null) return undefined;
    return entry.buffers;
  }

  set(rangeKey: string, channelKey: string, arrays: TelemArray[]): WebGLBuffer[] {
    const range = this.entries[rangeKey];
    if (range == null) {
      this.entries[rangeKey] = {};
    }
    const bufs = arrays.map((array) => {
      const buf = this.gl.createBuffer();
      if (buf == null) throw new UnexpectedError("failed to create buffer");
      this.gl.bindBuffer(this.gl.ARRAY_BUFFER, buf);
      this.gl.bufferData(this.gl.ARRAY_BUFFER, array.data.buffer, this.gl.STATIC_DRAW);
      return buf;
    });

    this.entries[rangeKey][channelKey] = new CacheEntry(bufs);
    return bufs;
  }
}

class CacheEntry {
  buffers: WebGLBuffer[];

  constructor(buffers: WebGLBuffer[]) {
    this.buffers = buffers;
  }

  delete(gl: WebGLRenderingContext): void {
    this.buffers.forEach((buffer) => gl.deleteBuffer(buffer));
  }
}
