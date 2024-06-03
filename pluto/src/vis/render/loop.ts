// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { type CanvasVariant } from "@/vis/render/context";

/**
 * An async function that executes the render in the loop. This function can return an
 * optional cleanup that is executed before the next render. This cleanup function will
 * be passed the signature of the previous render request.
 */
export type Func = () => Promise<Cleanup | undefined>;

/**
 * A request to render a component in the aether visualization tree. Submit a complete
 * version of this request to the {@link Loop} to render a component.
 */
export interface Request {
  /**
   * A key identifying the component requesting the render. This helps to prevent
   * duplicate renders for the same component from being executed.
   */
  key: string;
  /**
   * A priority ("high" or "low") for the render. High priority renders that have
   * an equal or greater number of canvases will replace low priority renders already
   * requested.
   */
  priority: Priority;
  /**
   * A list of canvases that the component is requesting to render to. This provides
   * information to cleanup functions about which canvases to clear. The component should
   * ONLY render to these canvases, otherwise the cleanup function may unnecessarily
   * clear canvases that should persist.
   */
  canvases: CanvasVariant[];
  /**
   * An async function that performs the render. This function can return an optional
   * cleanup that is executed before the next render. This cleanup function will be
   * passed the signature of the previous render request.
   */
  render: Func;
}

/**
 * A cleanup function that receives the request from the previous render. Cleanup
 * functions should clear canvases and other resources that need to be freed from
 * the previous render.
 */
export type Cleanup = (req: Request) => Promise<void>;

export type Priority = "high" | "low";

const PRIORITY_ORDER: Record<Priority, number> = { high: 1, low: 0 };

/**
 * Implements the core rendering loop for Synnax's aether components, accepting requests
 * into a queue and rendering them in sync with the browser animation frame.
 *
 * --------------------------------- VERY IMPORTANT ------------------------------
 *
 * This loop intentionally permits race conditions on the requests map access. We tried
 * locking this with an async mutex, but this resulted in a significant performance
 * hit for high-speed rendering for live telemetry. We've decided
 *
 */
export class Loop {
  private readonly mutex = new Mutex();
  /** Stores the current requests for rendering. */
  private readonly requests = new Map<string, Request>();
  /** Stores render cleanup functions for clearing canvases and other resources. */
  private readonly cleanup = new Map<string, Cleanup>();
  private readonly afterRender?: () => void;

  constructor(afterRender?: () => void) {
    void this.start();
    this.afterRender = afterRender;
  }

  /**
   * Sets a new request in the queue according to a set of rules:
   *
   * 1. If no request with the same key exists, add the request to the queue.
   * 2. If a request with the same key exists, replace it if the new request has a
   * greater or equal priority and a greater or equal number of canvases that are
   * being rendered to.
   *
   * @param req - The request to set.
   */
  async set(req: Request): Promise<void> {
    let releaser: (() => void) | undefined;
    if (req.priority === "high") {
      releaser = await this.mutex.acquire();
    }
    const existing = this.requests.get(req.key);
    if (existing == null) {
      this.requests.set(req.key, req);
    } else {
      const priorityOK =
        PRIORITY_ORDER[req.priority] >= PRIORITY_ORDER[existing.priority];
      const canvasesOK = req.canvases.length >= existing.canvases.length;
      if (priorityOK && canvasesOK) this.requests.set(req.key, req);
    }
    releaser?.();
  }

  /** Execute the render. */
  private async render(): Promise<void> {
    await this.mutex.runExclusive(async () => {
      const start = performance.now();
      if (this.requests.size === 0) return;
      const { requests: queue, cleanup } = this;
      for (const [k, f] of cleanup.entries()) {
        /** Execute all of our cleanup functions BEFORE we re-render. */
        const req = queue.get(k);
        if (req != null) {
          await f(req);
          cleanup.delete(k);
        }
      }
      /** Render components. */
      for (const req of queue.values()) {
        try {
          const cleanup = await req.render();
          // We're safe to set the cleanup function here because we know that req.key
          // is unique in the queue.
          if (cleanup != null) this.cleanup.set(req.key, cleanup);
        } catch (e) {
          console.error(e);
        }
      }
      const end = performance.now();
      const span = TimeSpan.milliseconds(end - start);
      if (span.greaterThan(TimeSpan.milliseconds(25))) {
        console.warn(
          `Render loop for ${this.requests.size} took longer than 16ms to execute: ${span.milliseconds}`,
        );
      }
      this.requests.clear();
      this.afterRender?.();
    });
  }

  /** Starts the rendering loop. */
  private async start(): Promise<void> {
    do {
      try {
        await this.render();
      } catch (e) {
        console.error(e);
      }
      await this.sleep();
    } while (true);
  }

  private async sleep(): Promise<number> {
    return await new Promise(requestAnimationFrame);
  }
}
