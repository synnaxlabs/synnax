// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";

/** Mirrors the private context key set by `render.Context`. Kept in sync with that
 * module's `CSS.B("render-context")` so the recorder lands under the key
 * `render.Context.use` reads. */
const RENDER_CONTEXT_KEY = CSS.B("render-context");

export const renderProviderStateZ = z.object({ context: z.any() });

/** Thin Composite that injects a `render.Context`-shaped value into the parent context
 * under the key consumed by `render.Context.use`. Lets a test recorder (or any
 * duck-typed render context) flow into the aether tree without a real `Canvas`
 * component, which depends on WebGL and OffscreenCanvas APIs jsdom does not provide. */
export class RenderProvider extends aether.Composite<typeof renderProviderStateZ> {
  static readonly TYPE = "render.test.RenderProvider";
  static readonly stateZ = renderProviderStateZ;
  schema = RenderProvider.stateZ;

  afterUpdate(ctx: aether.Context): void {
    ctx.set(RENDER_CONTEXT_KEY, this.state.context);
  }
}
