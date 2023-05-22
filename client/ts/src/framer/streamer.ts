// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errorZ, StreamClient } from "@synnaxlabs/freighter";
import { TimeStamp, UnparsedTimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

const reqZ = z.object({
  start: TimeStamp.z.optional(),
  keys: z.number().array(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  frame: frameZ,
  error: errorZ.optional(),
});

type Response = z.infer<typeof resZ>;

export class Streamer {
  private static readonly ENDPOINT = "/frame/read";
  private readonly stream: StreamProxy<Request, Response>;
  private readonly client: StreamClient;

  constructor(client: StreamClient) {
    this.client = client;
    this.stream = new StreamProxy("Streamer");
  }

  async open(start: UnparsedTimeStamp, keys: number[]): Promise<void> {
    // @ts-expect-error
    this.stream.stream = await this.client.stream(Streamer.ENDPOINT, reqZ, resZ);
    await this.stream.send({ start: new TimeStamp(start), keys });
  }

  async read(): Promise<Frame> {
    return new Frame((await this.stream.receive()).frame);
  }

  async update(keys: number[]): Promise<void> {
    this.stream.send({ keys });
  }
}
