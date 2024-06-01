// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type aether } from "@/aether/aether";
export type RequestF = (reason: string) => void;

export const REASON_LAYOUT = "layout";
export const REASON_DATA = "data";
export const REASON_TOOL = "tool";

export class Controller {
  f: RequestF;

  static readonly CONTEXT_KEY = "pluto-vis-renderer";

  private constructor(f: RequestF) {
    this.f = f;
  }

  static control(ctx: aether.Context, f: RequestF): void {
    ctx.set(Controller.CONTEXT_KEY, new Controller(f), false);
  }

  static useRequest(ctx: aether.Context): RequestF {
    return ctx.get<Controller>(Controller.CONTEXT_KEY).f;
  }

  static useOptionalRequest(ctx: aether.Context): RequestF | null {
    return ctx.getOptional<Controller>(Controller.CONTEXT_KEY)?.f ?? null;
  }

  static requestRender(ctx: aether.Context, reason: string): void {
    this.useRequest(ctx)(reason);
  }
}
