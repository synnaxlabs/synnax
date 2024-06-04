// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type xy } from "@synnaxlabs/x";

import { type CanvasVariant, type Context } from "@/vis/render/context";

export class Eraser {
  prevErase = box.ZERO;

  erase(
    ctx: Context,
    next: box.Box,
    prev: box.Box,
    overscan: xy.Crude,
    canvases: CanvasVariant[],
  ): void {
    if (!box.equals(prev, next) && !box.equals(this.prevErase, prev)) {
      ctx.erase(prev, overscan, ...canvases);
      this.prevErase = prev;
    } else ctx.erase(next, overscan, ...canvases);
  }
}
