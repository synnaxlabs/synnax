// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { query } from "@/query";

/**
 * One request in a batch: the endpoint it targets, such as `/group/create`, and the
 * payload that endpoint takes on its own. Build them with the domain clients, such as
 * `client.groups.createRequest` and `client.channels.setGroupRequest`.
 */
export interface Request<P = unknown> {
  endpoint: string;
  payload: P;
  /**
   * Writes the change the request makes to the client cache, before the Core answers.
   * Returns a rollback.
   */
  apply?: () => destructor.Destructor;
  /** Writes the response of the Core to the client cache, after the batch commits. */
  applyResponse?: (res: unknown) => void;
}

const sendReqZ = z.object({
  requests: z.object({ endpoint: z.string(), payload: z.unknown() }).array(),
});
const sendResZ = z.object({
  responses: z
    .unknown()
    .array()
    .default(() => []),
});

const applyAll = (requests: Request[]): destructor.Destructor[] =>
  requests.flatMap(({ apply }) => (apply == null ? [] : [apply()]));

/** Runs several write requests against the Core as one transaction. */
export class Client {
  private readonly unary: UnaryClient;

  constructor(unary: UnaryClient) {
    this.unary = unary;
  }

  /**
   * Runs the requests in order in one transaction. The first failure fails the whole
   * batch, and nothing commits. The cache writes of the requests apply before the batch
   * runs, apply again when it commits, and roll back when it fails. Then each request
   * writes its response to the cache.
   * @returns one response per request, in order.
   * @throws {ValidationError} if a request targets an endpoint the batch does not
   * accept, or if a request is invalid.
   */
  async send(requests: Request[], opts: query.WriteOptions = {}): Promise<unknown[]> {
    const wire = requests.map(({ endpoint, payload }) => ({ endpoint, payload }));
    const res = await query.optimistic({
      rollbacks: applyAll(requests),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.unary.send("/batch", { requests: wire }, sendReqZ, sendResZ),
    });
    applyAll(requests);
    requests.forEach(({ applyResponse }, i) => applyResponse?.(res.responses[i]));
    return res.responses;
  }
}
