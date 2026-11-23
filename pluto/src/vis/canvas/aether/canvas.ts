// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { box, runtime } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { line } from "@/vis/line/aether";
import { render } from "@/vis/render";

export const canvasStateZ = z.object({
  dpr: z.number(),
  region: box.box,
  bootstrap: z.boolean().default(false),
  bootstrapped: z.boolean().default(false),
  gpuMissing: z.boolean().default(false),
  // We set all of these to ay because nodejs can't identify OffscreenCanvas
  // and polyfilling causes more problems than it solves
  glCanvas: z.any().optional(),
  upper2dCanvas: z.any().optional(),
  lower2dCanvas: z.any().optional(),
  // Set in place of the canvases for the bitmap handoff, which posts frames to it.
  framePort: z.custom<MessagePort>().optional(),
  os: runtime.osZ,
});

const createCanvases = () => ({
  glCanvas: new OffscreenCanvas(0, 0),
  lower2dCanvas: new OffscreenCanvas(0, 0),
  upper2dCanvas: new OffscreenCanvas(0, 0),
});

export class Canvas extends aether.Composite<typeof canvasStateZ> {
  static readonly TYPE = "Canvas";

  override schema = canvasStateZ;
  renderCtx: render.Context | null = null;

  override afterUpdate(ctx: aether.Context): void {
    if (this.renderCtx == null) {
      if (!this.state.bootstrap) return;
      const { os, framePort } = this.state;
      const { glCanvas, lower2dCanvas, upper2dCanvas } =
        framePort == null ? this.state : createCanvases();
      if (glCanvas == null || lower2dCanvas == null || upper2dCanvas == null)
        throw new UnexpectedError(
          "[vis.worker.Canvas] - expected render context bootstrap to include all canvases",
        );
      this.renderCtx = render.Context.create(
        ctx,
        glCanvas as OffscreenCanvas,
        lower2dCanvas as OffscreenCanvas,
        upper2dCanvas as OffscreenCanvas,
        os,
        framePort,
      );
      if (this.renderCtx != null) line.Context.create(ctx, this.renderCtx);
      // Without a render context, no frame will ever use the port.
      else framePort?.close();
      this.setState((p) => ({
        ...p,
        bootstrap: false,
        bootstrapped: true,
        gpuMissing: this.renderCtx == null,
        glCanvas: undefined,
        lower2dCanvas: undefined,
        upper2dCanvas: undefined,
        framePort: undefined,
      }));
    } else this.renderCtx.update(ctx);
    this.renderCtx?.resize(this.state.region, this.state.dpr);
  }

  override afterDelete(): void {
    this.renderCtx?.close();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Canvas.TYPE]: Canvas,
};
