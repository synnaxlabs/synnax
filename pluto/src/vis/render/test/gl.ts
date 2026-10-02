// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Call } from "@/vis/render/test/Recorder";

/** Stand-in values for the WebGL enums that recorded calls are asserted against. */
export const GL_ENUMS = {
  ARRAY_BUFFER: 1,
  FLOAT: 2,
  UNSIGNED_BYTE: 3,
  STATIC_DRAW: 4,
  DYNAMIC_DRAW: 5,
  LINE_STRIP: 6,
  LINES: 7,
  VERTEX_SHADER: 8,
  FRAGMENT_SHADER: 9,
  COMPILE_STATUS: 10,
  TRIANGLES: 11,
};

const RETURNS: Record<string, () => unknown> = {
  createProgram: () => ({}),
  createShader: () => ({}),
  createBuffer: () => ({}),
  createVertexArray: () => ({}),
  getShaderParameter: () => true,
  getAttribLocation: () => 0,
  getUniformLocation: () => ({}),
};

export interface RecordingGL {
  gl: WebGL2RenderingContext;
  /** Every GL method call, in encounter order. */
  calls: Call[];
}

/** Creates a WebGL2 context that records every method call. Calls that return a handle
 * hand back a stub. */
export const createGL = (): RecordingGL => {
  const calls: Call[] = [];
  const target: Record<string, unknown> = { ...GL_ENUMS };
  const handler: ProxyHandler<Record<string, unknown>> = {
    get(t, prop) {
      if (typeof prop !== "string") return undefined;
      t[prop] ??= (...args: unknown[]): unknown => {
        calls.push({ op: prop, args });
        return RETURNS[prop]?.();
      };
      return t[prop];
    },
  };
  return { gl: new Proxy(target, handler) as unknown as WebGL2RenderingContext, calls };
};
