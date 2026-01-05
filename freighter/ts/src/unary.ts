// Copyright 2026 Synnax Labs, Inc.
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
 * An interface for an entity that implements a simple request-response
 * transport between two entities.
 */
export interface UnaryClient extends Transport {
  /**
   * Sends a request to the target server and waits until a response is received.
   * @param target - The target server to send the request to.
   * @param req - The request to send.
   * @param resSchema - The schema to validate the response against.
   */
  send: <RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    target: string,
    req: z.input<RQ> | z.infer<RQ>,
    reqSchema: RQ,
    resSchema: RS,
  ) => Promise<[z.infer<RS>, null] | [null, Error]>;
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

    async send<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
      target: string,
      req: z.input<RQ> | z.infer<RQ>,
      reqSchema: RQ,
      resSchema: RS,
    ): Promise<[z.infer<RS>, null] | [null, Error]> {
      const brk = new breaker.Breaker(cfg);
      do {
        const [res, err] = await this.wrapped.send(target, req, reqSchema, resSchema);
        if (err == null) return [res, null];
        if (!Unreachable.matches(err)) return [null, err];
        console.warn(`[freighter] ${brk.retryMessage}`, err);
        if (!(await brk.wait())) return [res, err];
      } while (true);
    }
  }
  return new WithBreaker(base);
};

export const sendRequired = async <RQ extends z.ZodType, RS extends z.ZodType = RQ>(
  client: UnaryClient,
  target: string,
  req: z.input<RQ> | z.infer<RQ>,
  reqSchema: RQ,
  resSchema: RS,
): Promise<z.infer<RS>> => {
  const [res, err] = await client.send(target, req, reqSchema, resSchema);
  if (err != null) throw err;
  return res;
};
