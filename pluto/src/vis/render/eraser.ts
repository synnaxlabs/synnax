// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, type XY } from "@synnaxlabs/x";

import { type Context } from "@/vis/render/context";

export class Eraser {
  prevErase = Box.ZERO;

  erase(ctx: Context, next: Box, prev: Box, overscan: XY): void {
    if (!prev.equals(next) && !this.prevErase.equals(prev)) {
      ctx.erase(prev, overscan);
      this.prevErase = prev;
    } else {
      ctx.erase(next, overscan);
    }
  }
}
