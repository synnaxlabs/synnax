// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { breaker } from "@synnaxlabs/x";
import { type z } from "zod";

import { Unreachable } from "@/errors";
import { type Middleware } from "@/middleware";
import { type Transport } from "@/transport";

/**
 * An interface for an entity that implements a simple request-response transport
 * between two entities.
 */
export interface UnaryClient extends Transport {
  /**
   * Sends a request to the target server and waits until a response is received.
   * @param target - The target server to send the request to.
   * @param req - The request to send.
   * @param reqSchema - The schema to validate the request against.
   * @param resSchema - The schema to validate the response against.
   */
  send<RQ extends z.ZodType>(
    target: string,
    req: z.input<RQ>,
    reqSchema: RQ,
  ): Promise<[Response, null] | [null, Error]>;
  send<RQ extends z.ZodType, RS extends z.ZodType>(
    target: string,
    req: z.input<RQ>,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<[z.infer<RS>, null] | [null, Error]>;
}

export const unaryWithBreaker = (
  base: UnaryClient,
  cfg: breaker.Config,
): UnaryClient => {
  class WithBreaker implements UnaryClient {
    readonly wrapped: UnaryClient;

    constructor(wrapper: UnaryClient) {
      this.wrapped = wrapper;
    }

    use(...mw: Middleware[]) {
      this.wrapped.use(...mw);
    }

    async send<RQ extends z.ZodType>(
      target: string,
      req: z.input<RQ>,
      reqSchema: RQ,
    ): Promise<[Response, null] | [null, Error]>;
    async send<RQ extends z.ZodType, RS extends z.ZodType>(
      target: string,
      req: z.input<RQ>,
      reqSchema: RQ,
      resSchema: RS,
    ): Promise<[z.infer<RS>, null] | [null, Error]>;
    async send<RQ extends z.ZodType, RS extends z.ZodType>(
      target: string,
      req: z.input<RQ>,
      reqSchema: RQ,
      resSchema?: RS,
    ): Promise<[Response, null] | [z.infer<RS>, null] | [null, Error]> {
      const brk = new breaker.Breaker(cfg);
      do {
        let err_: Error | null = null;
        if (resSchema == null) {
          const [res, err] = await this.wrapped.send(target, req, reqSchema);
          if (err == null) return [res, null];
          err_ = err;
        } else {
          const [res, err] = await this.wrapped.send(target, req, reqSchema, resSchema);
          if (err == null) return [res, null];
          err_ = err;
        }
        if (!Unreachable.matches(err_)) return [null, err_];
        console.warn(`[freighter] ${brk.retryMessage}`, err_);
        if (await brk.wait()) return [null, err_];
      } while (true);
    }
  }
  return new WithBreaker(base);
};

export interface SendRequired {
  <RQ extends z.ZodType>(
    client: UnaryClient,
    target: string,
    req: z.input<RQ>,
    reqSchema: RQ,
  ): Promise<Response>;
  <RQ extends z.ZodType, RS extends z.ZodType>(
    client: UnaryClient,
    target: string,
    req: z.input<RQ>,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<z.infer<RS>>;
}

export const sendRequired: SendRequired = async <
  RQ extends z.ZodType,
  RS extends z.ZodType,
>(
  client: UnaryClient,
  target: string,
  req: z.input<RQ>,
  reqSchema: RQ,
  resSchema?: RS,
): Promise<z.infer<RS> | Response> => {
  if (resSchema == null) {
    const [res, err] = await client.send(target, req, reqSchema);
    if (err != null) throw err;
    return res;
  }
  const [res, err] = await client.send(target, req, reqSchema, resSchema);
  if (err != null) throw err;
  return res;
};
