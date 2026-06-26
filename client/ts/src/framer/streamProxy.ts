// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream } from "@synnaxlabs/freighter";
import { errors } from "@synnaxlabs/x";
import { type z } from "zod";

import { UnexpectedError } from "@/errors";

export class StreamProxy<RQ extends z.ZodType, RS extends z.ZodType> {
  readonly name: string;
  private readonly stream: Stream<RQ, RS>;

  constructor(name: string, stream: Stream<RQ, RS>) {
    this.stream = stream;
    this.name = name;
  }

  async receive(): Promise<z.infer<RS>> {
    return await this.stream.receive();
  }

  received(): boolean {
    return this.stream.received();
  }

  async closeAndAck(): Promise<void> {
    this.stream.closeSend();
    while (true) {
      let res: z.infer<RS>;
      try {
        res = await this.stream.receive();
      } catch (err) {
        if (EOF.matches(err)) return;
        throw errors.fromUnknown(err);
      }
      throw new UnexpectedError(
        `${this.name} received unexpected response ${JSON.stringify(res)} on closure.`,
      );
    }
  }

  closeSend(): void {
    this.stream.closeSend();
  }

  send(req: z.input<RQ>): void {
    this.stream.send(req);
  }
}
