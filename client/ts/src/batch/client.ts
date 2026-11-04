// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

/**
 * One request in a batch: the endpoint it targets, such as `/group/create`, and the
 * payload that endpoint takes on its own. Build them with the domain builders, such as
 * `group.createRequest` and `channel.setGroupRequest`.
 */
export interface Request<P = unknown> {
  endpoint: string;
  payload: P;
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

/** Runs several write requests against the Core as one transaction. */
export class Client {
  private readonly unary: UnaryClient;

  constructor(unary: UnaryClient) {
    this.unary = unary;
  }

  /**
   * Runs the requests in order in one transaction. The first failure fails the whole
   * batch, and nothing commits. Cached records update from the Core's change stream.
   * @returns one response per request, in order.
   * @throws {ValidationError} if a request targets an endpoint the batch does not
   * accept, or if a request is invalid.
   */
  async send(requests: Request[]): Promise<unknown[]> {
    const res = await this.unary.send("/batch", { requests }, sendReqZ, sendResZ);
    return res.responses;
  }
}
