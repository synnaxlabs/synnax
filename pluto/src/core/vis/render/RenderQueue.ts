// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface RenderRequest {
  key: string;
  priority: RenderPriority;
  render: RenderFunction;
}
export type RenderFunction = () => Promise<RenderCleanup>;
export type RenderCleanup = () => Promise<void>;

export type RenderPriority = "high" | "low";

export class RenderQueue {
  queue: Record<string, RenderRequest>;
  cleanup: Record<string, RenderCleanup>;
  counter = 0;

  constructor() {
    this.queue = {};
    this.cleanup = {};

    void this.startRenderLoop();
  }

  async startRenderLoop(): Promise<void> {
    do {
      await this.render();
      await this.sleep();
    } while (true);
  }

  push(req: RenderRequest): void {
    const existing = this.queue[req.key];
    if (existing == null) {
      this.queue[req.key] = req;
      return;
    }
    this.queue[req.key] = req;
  }

  async sleep(): Promise<number> {
    return await new Promise(requestAnimationFrame);
  }

  async render(): Promise<void> {
    console.log(navigator.userAgent);
    this.counter += 1;
    const queue = this.queue;
    const cleanup = this.cleanup;
    this.queue = {};
    for (const [k, f] of Object.entries(cleanup)) {
      if (k in queue) {
        await f();
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
