// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type RenderFunction = () => Promise<void>;

export type RenderPriority = "high" | "low";

interface RenderEntry {
  render: RenderFunction;
  priority: RenderPriority;
}

export class RenderQueue {
  queue: Record<string, RenderEntry>;

  constructor() {
    this.queue = {};
    requestAnimationFrame(() => {
      void this.render();
    });
  }

  push(key: string, render: RenderFunction, priority: RenderPriority = "low"): void {
    const existing = this.queue[key];
    if (existing == null) {
      this.queue[key] = { render, priority };
      return;
    }
    if (existing.priority === "high" && priority !== "high") return;
    this.queue[key] = { render, priority };
  }

  async render(): Promise<void> {
    const queue = this.queue;
    this.queue = {};
    for (const { render } of Object.values(queue)) await render();
    requestAnimationFrame(() => {
      void this.render();
    });
  }
}
