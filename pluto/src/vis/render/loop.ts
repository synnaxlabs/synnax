// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";

import { type status } from "@/status/aether";
import { type CanvasVariant } from "@/vis/render/context";

/**
 * A function that executes the render in the loop. This function can return an
 * optional cleanup that is executed before the next render. This cleanup function will
 * be passed the signature of the previous render request.
 */
export interface Renderer {
  (): Cleanup | void;
}

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
  render: Renderer;
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
  handleError: status.ErrorHandler;
  afterRender?: () => void;
  instrumentation?: alamos.Instrumentation;
}

/**
 * Implements the core rendering loop for Synnax's aether components, accepting requests
 * into a queue and rendering them in sync with the browser animation frame.
 */
export class Loop {
  /** Stores the current requests for rendering. */
  private readonly requests = new Map<string, Request>();
  /** Stores render cleanup functions for clearing canvases and other resources. */
  private readonly cleanup = new Map<string, Cleanup>();
  /** A callback to run after each render call. */
  private readonly afterRender?: () => void;
  /** Instrumentation for logging, tracing, metrics, etc. */
  private readonly instrumentation: alamos.Instrumentation;
  /** A function to add status to the status bar. */
  private readonly handleError: status.ErrorHandler;

  constructor({
    afterRender,
    instrumentation = alamos.Instrumentation.NOOP,
    handleError,
  }: LoopArgs) {
    this.afterRender = afterRender;
    this.instrumentation = instrumentation;
    this.handleError = handleError;
    this.start();
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
    this.runCleanupsSync();
    this.renderSync();
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
        this.handleError(e, "render loop failed");
      }
    });
  }

  /** Starts the rendering loop. */
  private start(): void {
    const render = () => {
      try {
        this.render();
      } catch (e) {
        this.handleError(e, "render loop failed");
      }
      requestAnimationFrame(render);
    };
    requestAnimationFrame(render);
  }
}
