// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CanvasVariant } from "@/vis/render/context";

export type Func = () => Promise<Cleanup>;

export interface Request {
  key: string;
  priority: Priority;
  render: Func;
  canvases: CanvasVariant[];
}

export type Cleanup = (req: Request) => Promise<void>;

export type Priority = "high" | "low";

const PRIOTITY_ORDER: Record<Priority, number> = {
  high: 1,
  low: 0,
};

export class Queue {
  queue: Record<string, Request>;
  cleanup: Record<string, Cleanup>;

  constructor() {
    this.queue = {};
    this.cleanup = {};

    void this.startRenderLoop();
  }

  async startRenderLoop(): Promise<void> {
    do {
      try {
        await this.render();
      } catch (e) {
        console.error(e);
      }
      await this.sleep();
    } while (true);
  }

  push(req: Request): void {
    const existing = this.queue[req.key];
    if (existing == null) {
      this.queue[req.key] = req;
      return;
    }
    const priorityOK =
      PRIOTITY_ORDER[req.priority] >= PRIOTITY_ORDER[existing.priority];
    const canvasesOK = req.canvases.length >= existing.canvases.length;
    if (priorityOK && canvasesOK) this.queue[req.key] = req;
  }

  async sleep(): Promise<number> {
    return await new Promise(requestAnimationFrame);
  }

  async render(): Promise<void> {
    const queue = this.queue;
    const cleanup = this.cleanup;
    this.queue = {};
    for (const [k, f] of Object.entries(cleanup)) {
      if (k in queue) {
        await f(queue[k]);
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete cleanup[k];
      }
    }
    for (const { key, render } of Object.values(queue)) {
      const cleanup = await render();
      this.cleanup[key] = cleanup;
    }
  }
}
