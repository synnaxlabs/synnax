// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Rate } from "@synnaxlabs/x";

export type RenderFunction = () => Promise<void>;

export class RenderQueue {
  queue: Record<string, RenderFunction>;
  requested: boolean = false;

  constructor() {
    this.queue = {};
    setInterval(() => {
      if (Object.keys(this.queue).length === 0) return;
      void this.render();
    }, Rate.hz(60).period.milliseconds);
  }

  push(key: string, render: RenderFunction): void {
    this.queue[key] = render;
  }

  async render(): Promise<void> {
    const queue = this.queue;
    this.queue = {};
    const keys = Object.keys(queue);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      await queue[key]();
    }
    this.requested = false;
  }
}
