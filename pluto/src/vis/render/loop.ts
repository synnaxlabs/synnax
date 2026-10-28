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
 * Erases what a render drew. It receives the next request for the same key, narrowed to
 * the canvases of that request that the render drew on.
 */
export type Cleanup = (req: Request) => void;

interface PendingCleanup {
  cleanup: Cleanup;
  /** The canvases that the cleanup has not cleaned yet. */
  canvases: CanvasVariant[];
}

export type Priority = "high" | "low";

const PRIORITY_ORDER: Record<Priority, number> = { high: 1, low: 0 };

const add = (requests: Map<string, Request>, req: Request): void => {
  const existing = requests.get(req.key);
  if (existing == null) requests.set(req.key, req);
  else {
    const priorityOK =
      PRIORITY_ORDER[req.priority] >= PRIORITY_ORDER[existing.priority];
    const canvasesOK = req.canvases.length >= existing.canvases.length;
    if (priorityOK && canvasesOK) requests.set(req.key, req);
  }
};

interface LoopParams {
  handleError: status.ErrorHandler;
  afterRender?: () => void;
  instrumentation?: alamos.Instrumentation;
}

/**
 * Implements the main rendering loop for Synnax's aether components, accepting requests
 * into a queue and rendering them in sync with the browser animation frame.
 */
export class Loop {
  /** Stores the current requests for rendering. */
  private readonly requests = new Map<string, Request>();
  /** Stores the requests that draw on the GL canvas while the context is lost. */
  private readonly held = new Map<string, Request>();
  /** Stores render cleanup functions for clearing canvases and other resources. */
  private readonly cleanups = new Map<string, PendingCleanup[]>();
  /** A callback to run after each render call. */
  private readonly afterRender?: () => void;
  /** Instrumentation for logging, tracing, metrics, etc. */
  private readonly instrumentation: alamos.Instrumentation;
  /** A function to add status to the status bar. */
  private readonly handleError: status.ErrorHandler;
  /**
   * While true, the loop holds back afterRender and every request that draws on the GL
   * canvas. Set it while the WebGL context is lost.
   */
  glLost = false;

  constructor({
    afterRender,
    instrumentation = alamos.Instrumentation.NOOP,
    handleError,
  }: LoopParams) {
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
   */
  set(req: Request): void {
    add(this.requests, req);
  }

  /** Execute the render. */
  private render(): void {
    const { requests, held } = this;
    if (this.glLost)
      requests.forEach((req, key) => {
        if (!req.canvases.includes("gl")) return;
        add(held, req);
        requests.delete(key);
      });
    else if (held.size > 0) {
      requests.forEach((req) => add(held, req));
      held.forEach((req, key) => requests.set(key, req));
      held.clear();
    }
    if (requests.size === 0) return;
    this.runCleanupsSync();
    this.renderSync();
    requests.clear();
    if (!this.glLost) this.afterRender?.();
  }

  private runCleanupsSync(): void {
    const { cleanups, requests } = this;
    requests.forEach((req, key) => {
      const pending = cleanups.get(key);
      if (pending == null) return;
      const left: PendingCleanup[] = [];
      pending.forEach(({ cleanup, canvases }) => {
        const cleaned = canvases.filter((c) => req.canvases.includes(c));
        if (cleaned.length > 0) cleanup({ ...req, canvases: cleaned });
        const rest = canvases.filter((c) => !req.canvases.includes(c));
        if (rest.length > 0) left.push({ cleanup, canvases: rest });
      });
      if (left.length > 0) cleanups.set(key, left);
      else cleanups.delete(key);
    });
  }

  private renderSync() {
    /** Render components. */
    const { requests } = this;
    requests.forEach((req) => {
      try {
        const cleanup = req.render();
        if (cleanup != null)
          this.cleanups.set(req.key, [
            ...(this.cleanups.get(req.key) ?? []),
            { cleanup, canvases: req.canvases },
          ]);
      } catch (e) {
        this.handleError(e, "Failed to render");
      }
    });
  }

  /** Starts the rendering loop. */
  private start(): void {
    const render = () => {
      try {
        this.render();
      } catch (e) {
        this.handleError(e, "Failed to render");
      }
      requestAnimationFrame(render);
    };
    requestAnimationFrame(render);
  }
}
