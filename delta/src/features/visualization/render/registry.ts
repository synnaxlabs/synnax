// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Renderer } from "./renderer";

class RendererRegistry {
  private readonly renderers: Record<string, Renderer> = {};

  register(renderer: Renderer): void {
    this.renderers[renderer.type] = renderer;
  }

  get(type: string): Renderer {
    return this.renderers[type];
  }

  link(gl: WebGLRenderingContext): void {
    Object.values(this.renderers).forEach((r) => r.compile(gl));
  }
}

export const RENDERER_REGISTRY = new RendererRegistry();

RENDERER_REGISTRY.register();
