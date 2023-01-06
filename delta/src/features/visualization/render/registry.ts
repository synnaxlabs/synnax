// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LineRenderer } from "./line";
import { Renderer, RenderingContext } from "./renderer";

export class RendererRegistry {
  private readonly renderers: Record<string, Renderer<any>> = {};

  register<R>(renderer: Renderer<R>): void {
    this.renderers[renderer.type] = renderer;
  }

  get<R>(type: string): Renderer<R> {
    return this.renderers[type] as Renderer<R>;
  }

  async render(ctx: RenderingContext, request: any): Promise<void> {
    const renderer = this.get(request.renderer);
    if (renderer == null) throw new Error(`Unknown renderer: ${request.renderer}`);
    await renderer.render(ctx, request.request);
  }

  compile(gl: WebGLRenderingContext): void {
    Object.values(this.renderers).forEach((r) => r.compile(gl));
  }
}

export const RENDERER_REGISTRY = new RendererRegistry();

RENDERER_REGISTRY.register(new LineRenderer());
