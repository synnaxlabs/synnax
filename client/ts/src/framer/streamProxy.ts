// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream } from "@synnaxlabs/freighter";
import { type z } from "zod";

export class StreamProxy<RQ extends z.ZodType, RS extends z.ZodType> {
  readonly name: string;
  private readonly stream: Stream<RQ, RS>;

  constructor(name: string, stream: Stream<RQ, RS>) {
    this.stream = stream;
    this.name = name;
  }

  async receive(): Promise<z.infer<RS>> {
    const [res, err] = await this.stream.receive();
    if (err != null) throw err;
    return res;
  }

  received(): boolean {
    return this.stream.received();
  }

  async closeAndAck(): Promise<void> {
    this.stream.closeSend();
    while (true) {
      const [res, err] = await this.stream.receive();
      if (res != null)
        console.warn(
          `${this.name} received unexpected response on ${JSON.stringify(res)} closure.
        Please report this error to the Synnax team.`,
        );
      if (err != null) {
        if (EOF.matches(err)) return;
        throw err;
      }
    }
  }

  closeSend(): void {
    this.stream.closeSend();
  }

  send(req: z.input<RQ>): void {
    const err = this.stream.send(req);
    if (err != null) throw err;
  }
}
