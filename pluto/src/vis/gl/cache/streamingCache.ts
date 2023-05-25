import { LazyArray, Size } from "@synnaxlabs/x";

export interface GL {
  ARRAY_BUFFER: number;
  DYNAMIC_DRAW: number;
  bindBuffer: (target: number, buffer: WebGLBuffer | null) => void;
  bufferSubData: (
    target: number,
    dstByteOffset: number,
    srcData: ArrayBufferLike,
    srcOffset: number,
    length?: number
  ) => void;
  bufferData: (target: number, size: number, usage: number) => void;
  createBuffer: () => WebGLBuffer | null;
}

class Buffer {
  readonly len: Size;
  readonly cap: Size;
  readonly buf: WebGLBuffer;

  constructor(gl: GL, cap: Size) {
    const buf = gl.createBuffer();
    if (buf == null) throw new Error("failed to create buffer");
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, cap.valueOf(), gl.DYNAMIC_DRAW);
    this.len = Size.bytes(0);
    this.cap = cap;
    this.buf = buf;
  }

  get available(): Size {
    return this.len.sub(this.cap);
  }

  // Buffers the given array at the provided offset.
  // Returns the number of bytes buffered.
  buffer(gl: GL, arr: LazyArray, offset: Size): Size {
    const available = this.available;
    const numBytesBuffered = arr.size.sub(offset) < available ? arr.size : available;
    if (numBytesBuffered.isZero()) return numBytesBuffered;
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buf);
    gl.bufferSubData(
      gl.ARRAY_BUFFER,
      this.len.valueOf(),
      arr.data.buffer,
      arr.data.byteOffset + offset.valueOf(),
      numBytesBuffered.valueOf()
    );
    this.len = this.len.add(numBytesBuffered);
    return numBytesBuffered;
  }
}

export class GLStreamingCache {
  private readonly gl: GL;
  private readonly _buffers: Buffer[];
  private readonly bufSize: Size;

  constructor(gl: GL, bufSize: Size) {
    this.gl = gl;
    this._buffers = [];
    this.bufSize = bufSize;
  }

  update(arr: LazyArray): void {
    this._update(arr, Size.bytes(0));
  }

  private _update(arr: LazyArray, offset: Size): void {
    const n = this.curr.buffer(this.gl, arr, offset);
    if (n.isZero()) return;
    this.alloc();
    return this._update(arr, offset.add(n));
  }

  private get curr(): Buffer {
    return this._buffers[this._buffers.length - 1];
  }

  private alloc(): void {
    this._buffers.push(new Buffer(this.gl, this.bufSize));
  }

  get buffers(): WebGLBuffer[] {
    return this._buffers.map((b) => b.buf);
  }

  get len(): Size {
    return this._buffers.reduce((acc, b) => acc.add(b.len), Size.bytes(0));
  }

  get cap(): Size {
    return this._buffers.reduce((acc, b) => acc.add(b.cap), Size.bytes(0));
  }
}
