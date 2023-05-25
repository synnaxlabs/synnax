/**
 * GLBufferController is an interface for controlling the creation and buffering of
 * WebGLBuffers. It is used by GLDemandCache to create and manage WebGLBuffers.
 * It is implemented by WebGLRenderingContext.
 */
export interface GLBufferController {
  createBuffer: () => WebGLBuffer | null;
  bindBuffer: (target: number, buffer: WebGLBuffer | null) => void;
  bufferData: (
    target: number,
    sizeOrData: number | ArrayBufferLike,
    usage: number
  ) => void;
  bufferSubData: (
    target: number,
    offset: number,
    srcData: ArrayBufferLike,
    srcOffset?: number,
    length?: number
  ) => void;
  deleteBuffer: (buffer: WebGLBuffer) => void;
  getBufferParameter: (target: number, pname: number) => any;
  ARRAY_BUFFER: number;
  STATIC_DRAW: number;
  DYNAMIC_DRAW: number;
  BUFFER_SIZE: number;
}

export class MockGLBufferController implements GLBufferController {
  counter: number = 0;
  curr: number = 0;
  buffers: Map<number, ArrayBufferLike> = new Map();

  constructor() {
    this.buffers = new Map();
  }

  ARRAY_BUFFER: number = 0;
  STATIC_DRAW: number = 0;
  DYNAMIC_DRAW: number = 0;
  BUFFER_SIZE: number = 0;

  createBuffer(): WebGLBuffer | null {
    this.counter++;
    this.buffers.set(this.counter, new ArrayBuffer(0));
    return this.counter;
  }

  bindBuffer(target: number, buffer: WebGLBuffer | null): void {
    const b = this.buffers.get(buffer as number);
    if (b == null) throw new Error("Buffer not found");
    this.curr = buffer as number;
  }

  bufferData(
    target: number,
    sireOrData: number | ArrayBufferLike,
    usage: number
  ): void {
    if (typeof sireOrData === "number") {
      this.buffers.set(this.curr, new ArrayBuffer(sireOrData));
    } else {
      this.buffers.set(this.curr, sireOrData);
    }
  }

  bufferSubData(
    target: number,
    offset: number,
    srcData: ArrayBufferLike,
    srcOffset: number = 0,
    length: number = 0
  ): void {
    const b = this.buffers.get(this.curr);
    if (b == null) throw new Error("Buffer not found");
    const dst = new Uint8Array(b);
    const src = new Uint8Array(srcData);
    dst.set(src.slice(srcOffset, srcOffset + length), offset);
  }

  deleteBuffer(buffer: WebGLBuffer): void {
    this.buffers.delete(buffer as number);
  }

  getBufferParameter(target: number, pname: number): any {
    throw new Error("Method not implemented.");
  }
}
