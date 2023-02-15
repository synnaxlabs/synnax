// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { GLLineRenderer, LINE_RENDERER_TYPE } from "./line/renderer";
import { GLRenderer } from "./renderer";

export class GLRendererRegistry {
  private readonly renderers: Record<string, GLRenderer<any>> = {};

  register<R>(renderer: GLRenderer<R>): void {
    this.renderers[renderer.type] = renderer;
  }

  get<R>(type: string): GLRenderer<R> {
    return this.renderers[type] as GLRenderer<R>;
  }

  compile(gl: WebGLRenderingContext): void {
    Object.values(this.renderers).forEach((r) => r.compile(gl));
  }
}

export type DefaultRenderers = typeof LINE_RENDERER_TYPE;

export const newDefaultRendererRegistry = (): GLRendererRegistry => {
  const registry = new GLRendererRegistry();
  registry.register(new GLLineRenderer());
  return registry;
};
