// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ErrorPayloadSchema as errorPayload, Stream, StreamClient } from "@synnaxlabs/freighter";
import { TimeStamp, UnparsedTimeStamp } from "@synnaxlabs/x";
import { z } from "zod";
import { framePayload } from "@/framer/payload";
import { Frame, GeneralError, UnexpectedError } from "..";

const request = z.object({
  start: z.instanceof(TimeStamp).optional(),
  keys: z.number().array(),
})

type Request = z.infer<typeof request>;

const NOT_OPEN = new GeneralError(
  "FrameStreamReader.open() must be called before any other method"
)

const response = z.object({
  frame: framePayload,
  error: errorPayload.optional(),
})

type Response = z.infer<typeof response>;

export class FrameStreamReader {
  private static readonly ENDPOINT = "/frame/read";
  private stream: Stream<Request, Response> | undefined;
  private readonly client: StreamClient;

  constructor(client: StreamClient) {
    this.client = client;
  }

  async open(start: UnparsedTimeStamp, keys: number[]): Promise<void> {
    this.stream = await this.client.stream(
      FrameStreamReader.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-commnt
      // @ts-expect-error
      response,
    );
    await this.execute({ start: new TimeStamp(start), keys })
  }

  async read(): Promise<Frame> {
    if (this.stream == null) throw NOT_OPEN;
    const [res, exc] = await this.stream.receive();
    if (exc != null) throw exc;
    if (res == null)
      throw new UnexpectedError("received null response from stream reader")
    return new Frame(res.frame);
  }

  private async execute(request: Request) {
    if (this.stream == null) throw NOT_OPEN;
    this.stream.send(request)
  }

}
