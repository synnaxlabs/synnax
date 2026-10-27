// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** The WebGL context that a series buffers its data in, and its generation. */
export interface GLBufferController {
  readonly gl: {
    createBuffer: () => WebGLBuffer | null;
    bufferData: ((
      target: number,
      data: AllowSharedBufferSource,
      usage: number,
    ) => void) &
      ((target: number, size: number, usage: number) => void);
    bufferSubData: (
      target: number,
      offset: number,
      data: AllowSharedBufferSource,
    ) => void;
    bindBuffer: (target: number, buffer: WebGLBuffer | null) => void;
    deleteBuffer: (buffer: WebGLBuffer | null) => void;
    ARRAY_BUFFER: number;
    STATIC_DRAW: number;
    DYNAMIC_DRAW: number;
  };
  /**
   * Increments each time the context is restored after a loss. A restore destroys every
   * buffer made in an earlier generation.
   */
  readonly generation: number;
}

export type GLBufferUsage = "static" | "dynamic";

export const glBufferUsageZ = z.enum(["static", "dynamic"]);
