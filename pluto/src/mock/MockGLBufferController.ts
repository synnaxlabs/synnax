// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type GLBufferController } from "@synnaxlabs/x";
import { type Mock,vi } from "vitest";

export class MockGLBufferController implements GLBufferController {
  ARRAY_BUFFER: number = 1;
  STATIC_DRAW: number = 2;
  DYNAMIC_DRAW: number = 3;

  targets: Record<number, number> = {};
  counter: number = 0;
  buffers: Record<number, ArrayBuffer> = {};

  createBufferMock: Mock<[], WebGLBuffer | null> = vi.fn();
  bufferDataMock: Mock<[number, ArrayBufferLike | number, number]> = vi.fn();
  bufferSubDataMock: Mock<[number, number, ArrayBufferLike]> = vi.fn();
  bindBufferMock: Mock<[number, WebGLBuffer | null]> = vi.fn();
  deleteBufferMock: Mock<[WebGLBuffer | null]> = vi.fn();

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
    dataOrSize: ArrayBufferLike | number,
    usage: number,
  ): void {
    if (typeof dataOrSize === "number")
      this.buffers[this.targets[target]] = new ArrayBuffer(dataOrSize);
    else this.buffers[this.targets[target]] = dataOrSize;

    this.bufferDataMock(target, dataOrSize, usage);
  }

  bindBuffer(target: number, buffer: WebGLBuffer | null): void {
    if (buffer === 0) throw new Error("Cannot bind to 0");
    this.targets[target] = buffer as number;
    this.bindBufferMock(target, buffer);
  }

  bufferSubData(target: number, offset: number, data: ArrayBufferLike): void {
    let buffer = this.buffers[this.targets[target]];
    if (buffer == null) {
      buffer = new ArrayBuffer(offset + data.byteLength);
      this.buffers[target] = buffer;
    }
    const view = new Uint8Array(buffer);
    view.set(new Uint8Array(data), offset);
    this.bufferSubDataMock(target, offset, data);
  }
}
