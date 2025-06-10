// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { Mutex } from "async-mutex";

import { type CanvasVariant } from "@/vis/render/context";

/**
 * An async function that executes the render in the loop. This function can return an
 * optional cleanup that is executed before the next render. This cleanup function will
 * be passed the signature of the previous render request.
 */
export type Func = () => Cleanup | void;

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
export type Cleanup = (req: Request) => void;

export type Priority = "high" | "low";

const PRIORITY_ORDER: Record<Priority, number> = { high: 1, low: 0 };

interface LoopArgs {
  afterRender?: () => void;
  instrumentation?: alamos.Instrumentation;
}

/**
 * Implements the core rendering loop for Synnax's aether components, accepting requests
 * into a queue and rendering them in sync with the browser animation frame.
 *
 * --------------------------------- VERY IMPORTANT ------------------------------
 *
 * This loop intentionally permits race conditions under render request priorities that
 * are not 'high'. This ensures smooth rendering performance on low priority cycles
 * (such as new data), while guarantees proper rendering on high priority cycles
 * (such as the user altering the layout)).
 */
export class Loop {
  /** To lock things so renders don't collide */
  private readonly mutex = new Mutex();
  /** Stores the current requests for rendering. */
  private readonly requests = new Map<string, Request>();
  /** Stores render cleanup functions for clearing canvases and other resources. */
  private readonly cleanup = new Map<string, Cleanup>();
  /** A callback to run after each render call. */
  private readonly afterRender?: () => void;
  /** Instrumentation for logging, tracing, metrics, etc. */
  private readonly instrumentation: alamos.Instrumentation;
  /** The total number of renders */
  private count = 0;

  constructor({
    afterRender,
    instrumentation = alamos.Instrumentation.NOOP,
  }: LoopArgs) {
    this.afterRender = afterRender;
    this.instrumentation = instrumentation;
    void this.start();
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
  set(req: Request): void {
    const existing = this.requests.get(req.key);
    if (existing == null) this.requests.set(req.key, req);
    else {
      const priorityOK =
        PRIORITY_ORDER[req.priority] >= PRIORITY_ORDER[existing.priority];
      const canvasesOK = req.canvases.length >= existing.canvases.length;
      if (priorityOK && canvasesOK) this.requests.set(req.key, req);
    }
  }

  /** Execute the render. */
  private render(): void {
    const { requests } = this;
    if (requests.size === 0) return;
    const endCycle = this.instrumentation.T.bench("render-cycle");
    const endCleanup = this.instrumentation.T.bench("render-cycle-cleanup");
    this.runCleanupsSync();
    endCleanup();
    const endRender = this.instrumentation.T.bench("render-cycle-render");
    this.renderSync();
    endRender();
    endCycle();
    this.requests.clear();
    this.afterRender?.();
  }

  private runCleanupsSync(): void {
    const { cleanup, requests } = this;
    cleanup.forEach((f, k) => {
      /** Execute all of our cleanup functions BEFORE we re-render. */
      const req = requests.get(k);
      if (req != null) {
        f(req);
        cleanup.delete(k);
      }
    });
  }

  private renderSync() {
    /** Render components. */
    const { requests } = this;
    requests.forEach((req) => {
      try {
        const cleanup = req.render();
        // We're safe to set the cleanup function here because we know that req.key
        // is unique in the queue.
        if (cleanup != null) this.cleanup.set(req.key, cleanup);
      } catch (e) {
        console.error(e);
      }
    });
  }

  /** Starts the rendering loop. */
  private async start(): Promise<void> {
    const render = () => {
      try {
        this.render();
      } catch (e) {
        console.error(e);
      }
      requestAnimationFrame(render);
    };
    requestAnimationFrame(render);
  }
}
