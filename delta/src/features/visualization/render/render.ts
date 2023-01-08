// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY } from "@synnaxlabs/pluto";

import { TelemetryClient } from "../telem/client";

import { Compiler } from "./compiler";
import { LineRenderer, LINE_RENDERER_TYPE } from "./line";
import { ScissoredRenderer } from "./scissor";

export type RenderingUnits = "px" | "decimal";

/**
 * Context provided to a renderer that contains all necessary tools and information for
 * rendering.
 *
 * IMPORTANT NOTE: The provided root scaling and offsets assume a different coordinate
 * space than a traditional webgl clip. The coordinate space is 0 to 1, where 0 is
 * the bottom left corner.
 *
 * @property gl - The WebGL context.
 *
 * @property rootScaleClip - The root scale provided by the canvas in clip space. This scale
 * should be applied after all other scales, and is used to 'sub-render' to a specific
 * region of the canvas. i.e. u_scale_root * (VERTEX_POSITION_AFTER_EVERYTHING) + u_offset_root
 *
 * @property rootOffsetClip - The root offset provided by the canvas in clip space. This
 * offset should be applied after all other offsets, and is used to 'sub-render' to a specific
 * region of the canvas. i.e. u_scale_root * (VERTEX_POSITION_AFTER_EVERYTHING) + u_offset_root
 *
 * @property rootOffsetPx - The root offset provided by the canvas in pixel space. This
 * is measured from the bottom left corner of the canvas.
 *
 * @property dpr - The device pixel ratio of the canvas.
 *
 * @property aspect - The aspect ratio of the canvas.
 */
export interface RenderingContext {
  /* The WebGL context. */
  readonly gl: WebGLRenderingContext;
  scale: (box: Box) => XY;
  offset: (box: Box, units: RenderingUnits) => XY;
  refreshCanvas: () => void;
  readonly dpr: number;
  readonly aspect: number;
  readonly client: TelemetryClient;
  readonly registry: RendererRegistry;
}

/**
 * A renderer for a specific type of entity. A renderer should not maintain any internal
 * state relating to specific entities, but should instead rely on the request properties
 * to determine how to render it.
 */
export interface Renderer<R> extends Compiler {
  /** Type is a unique type for the renderer. */
  type: string;
  /** Renders the given entity under the RenderingContext.  */
  render: (ctx: RenderingContext, req: R) => Promise<void>;
}

export class RendererRegistry {
  private readonly renderers: Record<string, Renderer<any>> = {};

  register<R>(renderer: Renderer<R>): void {
    this.renderers[renderer.type] = renderer;
  }

  get<R>(type: string): Renderer<R> {
    return this.renderers[type] as Renderer<R>;
  }

  compile(gl: WebGLRenderingContext): void {
    Object.values(this.renderers).forEach((r) => r.compile(gl));
  }
}

export type DefaultRenderers = typeof LINE_RENDERER_TYPE;

export const newDefaultRendererRegistry = (): RendererRegistry => {
  const registry = new RendererRegistry();
  registry.register(new ScissoredRenderer(new LineRenderer(), true, { x: 24, y: 48 }));
  return registry;
};
