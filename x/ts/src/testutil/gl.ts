// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";

import { type GLBufferController } from "@/telem";

// WebGL copies data whether it is a buffer or a view.
const bytes = (data: AllowSharedBufferSource): Uint8Array =>
  ArrayBuffer.isView(data)
    ? new Uint8Array(data.buffer, data.byteOffset, data.byteLength)
    : new Uint8Array(data);

/** A controller that is also the GL context it hands to a series. */
export class MockGLBufferController implements GLBufferController {
  readonly gl: GLBufferController["gl"] = this;
  generation: number = 0;
  ARRAY_BUFFER: number = 1;
  STATIC_DRAW: number = 2;
  DYNAMIC_DRAW: number = 3;

  targets: Record<number, number> = {};
  counter: number = 0;
  buffers: Record<number, ArrayBuffer> = {};

  createBufferMock: Mock<() => WebGLBuffer | null> = vi.fn();
  bufferDataMock: Mock<(a: number, b: ArrayBufferLike | number, c: number) => void> =
    vi.fn();
  bufferSubDataMock: Mock<(a: number, b: number, c: ArrayBufferLike) => void> = vi.fn();
  bindBufferMock: Mock<(a: number, b: WebGLBuffer | null) => void> = vi.fn();
  deleteBufferMock: Mock<(a: WebGLBuffer | null) => void> = vi.fn();

  deleteBuffer(buffer: WebGLBuffer | null): void {
    if (buffer != null) delete this.buffers[buffer as number];
    this.deleteBufferMock(buffer);
  }

  createBuffer(): WebGLBuffer | null {
    this.createBufferMock();
    const v = ++this.counter;
    this.buffers[v] = new ArrayBuffer(0);
    return v;
  }

  bufferData(
    target: number,
    dataOrSize: AllowSharedBufferSource | number,
    usage: number,
  ): void {
    if (typeof dataOrSize === "number")
      this.buffers[this.targets[target]] = new ArrayBuffer(dataOrSize);
    else this.buffers[this.targets[target]] = bytes(dataOrSize).slice().buffer;

    this.bufferDataMock(target, dataOrSize as ArrayBuffer, usage);
  }

  bindBuffer(target: number, buffer: WebGLBuffer | null): void {
    if (buffer === 0) throw new Error("Cannot bind to 0");
    this.targets[target] = buffer as number;
    this.bindBufferMock(target, buffer);
  }

  bufferSubData(target: number, offset: number, data: AllowSharedBufferSource): void {
    let buffer = this.buffers[this.targets[target]];
    if (buffer == null) {
      buffer = new ArrayBuffer(offset + data.byteLength);
      this.buffers[target] = buffer;
    }
    const view = new Uint8Array(buffer);
    view.set(bytes(data), offset);
    this.bufferSubDataMock(target, offset, data as ArrayBuffer);
  }
}
