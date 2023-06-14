// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { VFactory, VComposite } from "@/core/virtual/worker";
import { Value } from "@/core/vis/pid/Value/worker";
import { RenderContext, RenderQueue } from "@/core/vis/render";
import { TelemProvider } from "@/core/vis/telem/TelemService";

export const pidState = z.object({
  position: XY.z,
  region: Box.z,
});

export type PIDState = z.input<typeof pidState>;
export type ParsedPIDState = z.output<typeof pidState>;

export class PIDFactory implements VFactory<PID> {
  ctx: RenderContext;
  telem: TelemProvider;
  renderQueue: RenderQueue;

  constructor(ctx: RenderContext, telem: TelemProvider, renderQueue: RenderQueue) {
    this.ctx = ctx;
    this.telem = telem;
    this.renderQueue = renderQueue;
  }

  create(type: string, key: string, state: any): PID {
    return new PID(key, this.ctx, this.telem, this.renderQueue, state);
  }
}

export type PIDItem = Value;

export class PIDItemFactory implements VFactory<PIDItem> {
  ctx: RenderContext;
  telem: TelemProvider;
  requestRender: () => void;

  constructor(ctx: RenderContext, telem: TelemProvider, requestRender: () => void) {
    this.requestRender = requestRender;
    this.ctx = ctx;
    this.telem = telem;
  }

  create(type: string, key: string, state: any): PIDItem {
    switch (type) {
      case "value":
        return new Value(key, this.ctx, state, this.telem, this.requestRender);
      default:
        throw new Error(`Unknown PID item type: ${type}`);
    }
  }
}

export class PID extends VComposite<PIDState, ParsedPIDState, PIDItem> {
  static readonly TYPE = "pid";

  ctx: RenderContext;
  renderQueue: RenderQueue;

  constructor(
    key: string,
    ctx: RenderContext,
    telem: TelemProvider,
    renderQueue: RenderQueue,
    state: any
  ) {
    const factory = new PIDItemFactory(ctx, telem, () => this.requestRender());
    super(PID.TYPE, key, factory, pidState, state);
    this.ctx = ctx;
    this.renderQueue = renderQueue;
    this.bindStateHook(() => this.requestRender());
  }

  async render(): Promise<void> {
    this.ctx.erase(new Box(this.state.region));
    this.children.forEach((child) => child.render({ position: this.state.position }));
  }

  private requestRender(): void {
    this.renderQueue.push(this.key, async () => await this.render());
  }
}
