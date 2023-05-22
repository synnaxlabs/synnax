// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* eslint-disable @typescript-eslint/no-throw-literal */
import type { StreamClient } from "@synnaxlabs/freighter";
import { decodeError, errorZ } from "@synnaxlabs/freighter";
import {
  NativeTypedArray,
  LazyArray,
  TimeStamp,
  UnparsedTimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { ForwardFrameAdapter } from "./adapter";

import { ChannelKeyOrName, ChannelParams } from "@/channel/payload";
import { ChannelRetriever } from "@/channel/retriever";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

enum Command {
  Open = 0,
  Write = 1,
  Commit = 2,
  Error = 3,
}

const configZ = z.object({
  start: TimeStamp.z,
  keys: z.number().array().optional(),
});

const reqZ = z.object({
  command: z.nativeEnum(Command),
  config: configZ.optional(),
  frame: frameZ.optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  ack: z.boolean(),
  command: z.nativeEnum(Command),
  error: errorZ.optional().nullable(),
});

type Response = z.infer<typeof resZ>;

/**
 * Writer is used to write telemetry to a set of channels in time order.
 * It should not be instantiated directly, and should instead be instantited via the
 * FramerClient {@link FrameClient#openWriter}.
 *
 * The writer is a streaming protocol that is heavily optimized for prerformance. This
 * comes at the cost of icnreased complexity, and should only be used directly when
 * writing large volumes of data (such as recording telemetry from a sensor or ingsting
 * data froma file). Simpler methods (such as the frame client's write method) should
 * be used for most use cases.
 *
 * The protocol is as follows:
 *
 * 1. The writer is opened with a starting timestamp and a list of channel keys. The
 * writer will fail to open if hte starting timstamp overlaps with any existing telemetry
 * for any channels specified. If the writer opens successfuly, the caller is then
 * free to write frames to the writer.
 *
 * 2. To write a frame, the caller can use the write method and follow the validation
 * rules described in its method's documentation. This process is asynchronous, meaning
 * that write calls may return before teh frame has been written to the cluster. This
 * also means that the writer can accumulate an error after write is called. If the writer
 * accumulates an erorr, all subsequent write and commit calls will return False. The
 * caller can check for errors by calling the error mehtod, which returns the accumulated
 * error and resets the writer for future use. The caller can also check for errors by
 * closing the writer, which will throw any accumulated error.
 *
 * 3. To commit the written frames to the cluster, the caller can call the commit method.
 * Unlike write, commit is synchronous, meaning that it will not return until the frames
 * have been written to the cluster. If the writer has accumulated an erorr, commit will
 * return false. After the caller acknowledges the erorr, they can attempt to commit again.
 * Commit can be called several times throughout a writer's lifetime, and will only
 * commit the frames that have been written since the last commit.
 *
 * 4. A writer MUST be closed after use in order to prevent resource leaks. Close should
 * typically be called in a 'finally' block. If the writer has accumulated an error,
 * close will throw the error.
 */
export class Writer {
  private static readonly ENDPOINT = "/frame/write";
  private readonly client: StreamClient;
  private readonly stream: StreamProxy<Request, Response>;
  private readonly channels: ChannelRetriever;
  private adapter: ForwardFrameAdapter;

  constructor(client: StreamClient, retriever: ChannelRetriever) {
    this.client = client;
    this.stream = new StreamProxy("Writer");
    this.adapter = new ForwardFrameAdapter();
    this.channels = retriever;
  }

  /**
   * Opens the writer to wirte a range of telemetry starting at the given time.
   *
   * @param start -  The starting timestamp of the new range to write to. If start overlaps
   * with existing telemetry, the writer will fail to open.
   *
   * @param keys - A list of keys representing the channels the writer will write to. All
   * frames written to the writer must have channel keys in this list.
   */
  async _open(start: UnparsedTimeStamp, ...channels: ChannelParams[]): Promise<void> {
    this.adapter = await ForwardFrameAdapter.fromParams(this.channels, ...channels);
    // @ts-expect-error
    this.stream.stream = await this.client.stream(Writer.ENDPOINT, reqZ, resZ);
    await this.execute({
      command: Command.Open,
      config: { start: new TimeStamp(start), keys: this.adapter.keys },
    });
  }

  async write(channel: ChannelKeyOrName, data: NativeTypedArray): Promise<boolean>;

  async write(frame: Frame): Promise<boolean>;

  /**
   * Writes the given frame to the database.
   *
   * @param frame - The frame to write to the database. The frame must:
   *
   *    1. Have exactly one array for each key in the list of keys provided to the
   *    writer's open method.
   *    2. Have equal length arrays for each key.
   *    3. When writing to an index (i.e. TimeStamp) channel, the values must be
   *    monotonically increasing.
   *
   * @returns false if the writer has accumulated an error. In this case, the caller
   * should acknowledge the error by calling the error method or closing the writer.
   */
  async write(
    frame: Frame | ChannelKeyOrName,
    data?: NativeTypedArray
  ): Promise<boolean> {
    if (!(frame instanceof Frame)) {
      frame = new Frame(new LazyArray(data as NativeTypedArray), frame);
    }
    frame = this.adapter.adapt(frame);
    if (this.errorAccumulated) return false;
    this.stream.send({ command: Command.Write, frame: frame.toPayload() });
    return true;
  }

  /**
   * Commits the written frames to the database. Commit is synchronous, meaning that it
   * will not return until all frames have been commited to the database.
   *
   * @returns false if the commit failed due to an error. In this case, the caller
   * should acknowledge the error by calling the error method or closing the writer.
   * After the caller acknowledges the error, they can attempt to commit again.
   */
  async commit(): Promise<boolean> {
    if (this.errorAccumulated) return false;
    const res = await this.execute({ command: Command.Commit });
    return res.ack;
  }

  /**
   * @returns  The accumulated error, if any. This method will clear the writer's error
   * state, allowing the writer to be used again.
   */
  async error(): Promise<Error | null> {
    this.stream.send({ command: Command.Error });
    const res = await this.execute({ command: Command.Error });
    return res.error != null ? decodeError(res.error) : null;
  }

  /**
   * Closes the writer, raising any accumulated error encountered during operation.
   * A writer MUST be closed after use, and this method should probably be placed
   * in a 'finally' block.
   */
  async close(): Promise<void> {
    await this.stream.closeAndAck();
  }

  async execute(req: Request): Promise<Response> {
    this.stream.send(req);
    while (true) {
      const res = await this.stream.receive();
      if (res.command === req.command) return res;
      console.warn("writer received unexpected response", res);
    }
  }

  private get errorAccumulated(): boolean {
    return this.stream.received();
  }
}
