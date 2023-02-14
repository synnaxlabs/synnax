// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* eslint-disable @typescript-eslint/no-throw-literal */
import type { Stream, StreamClient } from "@synnaxlabs/freighter";
import { decodeError, EOF, ErrorPayloadSchema } from "@synnaxlabs/freighter";
import {
  NativeTypedArray,
  LazyArray,
  TimeStamp,
  UnparsedTimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { GeneralError } from "@/errors";

import { Frame } from "./frame";
import { framePayloadSchema } from "./payload";

enum Command {
  None = 0,
  Write = 1,
  Commit = 2,
  Error = 3,
}

const configSchema = z.object({
  start: z.instanceof(TimeStamp).optional(),
  keys: z.string().array().optional(),
});

const requestSchema = z.object({
  command: z.nativeEnum(Command),
  config: configSchema.optional(),
  frame: framePayloadSchema.optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  ack: z.boolean(),
  command: z.nativeEnum(Command),
  error: ErrorPayloadSchema.optional(),
});

type Response = z.infer<typeof responseSchema>;

const NOT_OPEN = new GeneralError(
  "Writer has not been opened. Please open before calling write() or close()."
);

/**
 * CoreWriter is used to write a range of telemetry to a set of channels in time order.
 * It should not be instantiated directly, and should instead be instantited via the
 * FramerClient {@link FramerClient#openWriter}.
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
export class FrameWriter {
  private static readonly ENDPOINT = "/frame/write";
  private readonly client: StreamClient;
  private stream: Stream<Request, Response> | undefined;

  constructor(client: StreamClient) {
    this.client = client;
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
  async open(start: UnparsedTimeStamp, keys: string[]): Promise<void> {
    this.stream = await this.client.stream(
      FrameWriter.ENDPOINT,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      requestSchema,
      responseSchema
    );
    this.stream.send({
      command: Command.None,
      config: { start: new TimeStamp(start), keys },
    });
    const [, err] = await this.stream.receive();
    if (err != null) throw err;
  }

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
  async write(frame: Frame): Promise<boolean> {
    if (this.stream == null) throw NOT_OPEN;
    if (this.stream.received()) return false;

    const err = this.stream.send({ command: Command.Write, frame: frame.toPayload() });
    if (err != null) throw err;
    return true;
  }

  async writeArray(key: string, data: NativeTypedArray): Promise<boolean> {
    return await this.write(new Frame(new LazyArray(data), key));
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
    if (this.stream == null) throw NOT_OPEN;
    if (this.stream.received()) return false;

    const err = this.stream.send({ command: Command.Commit });
    if (err != null) throw err;

    while (true) {
      const [res, err] = await this.stream.receive();
      if (err != null) throw err;
      if (res != null && res?.command === Command.Commit) return res.ack;
      this.warnUnexpectedResponse(res);
    }
  }

  /**
   * @returns  The accumulated error, if any. This method will clear the writer's erorr
   * state, allowing the writer to be used again.
   */
  async error(): Promise<Error | undefined> {
    if (this.stream == null) throw NOT_OPEN;

    const err = this.stream.send({ command: Command.Error });
    if (err != null) throw err;

    while (true) {
      const [res, err] = await this.stream.receive();
      if (err != null) throw err;
      if (res != null && res?.command === Command.Error && res.error != null)
        return decodeError(res.error);
      this.warnUnexpectedResponse(res);
    }
  }

  /**
   * Closes the writer, raising any accumulated error encountered during operation.
   * A writer MUST be closed after use, and this method should probably be placed
   * in a 'finally' block.
   */
  async close(): Promise<void> {
    if (this.stream == null) throw NOT_OPEN;
    this.stream.closeSend();
    const [res, err] = await this.stream.receive();
    if (err == null && res?.error != null) throw decodeError(res.error);
    if (!(err instanceof EOF)) throw err;
  }

  private warnUnexpectedResponse(res?: Response): void {
    if (res == null) console.warn("writer received unexpected null response");
    console.warn("writer received unexpected response", res);
  }
}
