// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Context is the metadata associated with a freighter transport request.
 *
 * @property target - The target the request is being issued to.
 * @property protocol - The protocol used to issue the request.
 * @property params - Arbitrary string parameters that can be set by client side
 *   middleware and read by server side middleware.
 */
export interface Context {
  target: string;
  role: Role;
  protocol: string;
  params: Record<string, string>;
}

export const ROLES = ["client", "server"] as const;
export type Role = (typeof ROLES)[number];

/** Next executes the next middleware in the chain. */
export type Next = (ctx: Context) => Promise<[Context, Error | null]>;

/**
 * Middleware represents a general middleware function that can be used to
 * parse/attach metadata to a request or alter its behavior.
 */
export type Middleware = (ctx: Context, next: Next) => Promise<[Context, Error | null]>;

/**
 * Finalizer is a middleware that is executed as the last step in the chain.
 * Finalizer middleware should be used to execute the request.
 */
type Finalizer = (ctx: Context) => Promise<[Context, Error | null]>;

/**
 * MiddlewareCollector is a class that can be used to collect and execute
 * middleware in order to implement the Transport interface.
 */
export class MiddlewareCollector {
  middleware: Middleware[] = [];

  /** Implements the Transport interface */
  use(...mw: Middleware[]): void {
    this.middleware.push(...mw);
  }

  /**
   * Executes middleware in order, passing the the metadata to each middleware
   * until the end of the chain is reached. It then calls the finalizer with the
   * metadata.
   *
   * @param ctx - The context to pass to the middleware.
   * @param finalizer - The finalizer to call with the metadata.
   * @returns An error if one was encountered, otherwise undefined.
   */
  async executeMiddleware(
    ctx: Context,
    finalizer: Finalizer,
  ): Promise<[Context, Error | null]> {
    let i = 0;
    const next = async (md: Context): Promise<[Context, Error | null]> => {
      if (i === this.middleware.length) return await finalizer(md);
      const _mw = this.middleware[i];
      i++;
      return await _mw(md, next);
    };
    return await next(ctx);
  }
}
