// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LazyArray, DemandCache, DemandCacheEntry, Size, KV } from "@synnaxlabs/x";

interface GLCacheBuffer {
  buf: WebGLBuffer;
  size: Size;
}

export type GLDemandCacheEntry = DemandCacheEntry<string, GLCacheBuffer[]>;

/**
 * GLDemandCache controls the creation and buffering of WebGLBuffers. It maintains a
 * soft memory limit by garbage collecting buffers that are not in use. To maintain
 * the memory limit, it's essential that the caller calls `release` on the returned
 * DemandCacheEntry when the buffers are no longer needed.
 */
export class GLDemandCache
  implements KV<string, GLDemandCacheEntry, string, LazyArray[]>
{
  private readonly gl: GLBufferController;
  private readonly internal: DemandCache<string, GLCacheBuffer[]>;
  softLimit: Size = Size.megabytes(500);
  size: Size = Size.bytes(0);

  constructor(gl: GLBufferController) {
    this.gl = gl;
    this.internal = new DemandCache();
  }

  get(key: string): GLDemandCacheEntry | null {
    this.gc();
    return this.internal.get(key);
  }

  set(key: string, arrays: LazyArray[]): void {
    this.gc();
    this.internal.set(
      key,
      arrays.map((a) => this.createBuffer(a))
    );
  }

  delete(key: string): void {
    this.gc();
    const entry = this.internal.get(key);
    if (entry == null) return;
    this._delete(entry);
  }

  private _delete(entry: GLDemandCacheEntry): void {
    entry.value.forEach((buf) => {
      this.gl.bindBuffer(this.gl.ARRAY_BUFFER, null);
      this.size = this.size.sub(buf.size);
      this.gl.deleteBuffer(buf);
    });
    this.internal.delete(entry.key);
  }

  private gc(): void {
    if (this.size.smallerThan(this.softLimit)) return;
    const entries = this.internal.getDemandUnder(1);
    entries.forEach((entry) => this.delete(entry.key));
  }

  private createBuffer(arr: LazyArray): GLCacheBuffer {
    const buf = this.gl.createBuffer();
    if (buf == null) throw new Error("failed to create buffer");
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, buf);
    this.gl.bufferData(this.gl.ARRAY_BUFFER, arr.data.buffer, this.gl.DYNAMIC_DRAW);
    const size = new Size(
      this.gl.getBufferParameter(this.gl.ARRAY_BUFFER, this.gl.BUFFER_SIZE)
    );
    this.size = this.size.add(size);
    return { buf, size };
  }
}
