// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream, type WebSocketClient } from "@synnaxlabs/freighter";
import { array, control, errors } from "@synnaxlabs/x";
import {
  type CrudeSeries,
  type CrudeTimeStamp,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x/telem";
import { z } from "zod/v4";

import { channel } from "@/channel";
import { SynnaxError } from "@/errors";
import { WriteAdapter } from "@/framer/adapter";
import { WSWriterCodec } from "@/framer/codec";
import { type CrudeFrame, frameZ } from "@/framer/frame";

export enum WriterCommand {
  Open = 0,
  Write = 1,
  Commit = 2,
  SetAuthority = 3,
}

export enum WriterMode {
  PersistStream = 1,
  Persist = 2,
  Stream = 3,
}

export type CrudeWriterMode = "persist" | "stream" | "persistStream" | WriterMode;

const constructWriterMode = (mode: CrudeWriterMode): WriterMode => {
  switch (mode) {
    case "persist":
      return WriterMode.Persist;
    case "stream":
      return WriterMode.Stream;
    case "persistStream":
      return WriterMode.PersistStream;
    default:
      if (typeof mode === "number" && mode in WriterMode) return mode;
      throw new Error(`invalid writer mode: ${mode}`);
  }
};

export const ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT: TimeSpan = new TimeSpan(-1);

export class WriterClosedError extends SynnaxError.sub("writer_closed") {
  constructor() {
    super("WriterClosed");
  }
}

const netConfigZ = z.object({
  start: TimeStamp.z.optional(),
  controlSubject: control.subjectZ.optional(),
  keys: channel.keyZ.array().optional(),
  authorities: control.authorityZ.array().optional(),
  mode: z.enum(WriterMode).optional(),
  errOnUnauthorized: z.boolean().optional(),
  enableAutoCommit: z.boolean().optional(),
  autoIndexPersistInterval: TimeSpan.z.optional(),
});

interface Config extends z.infer<typeof netConfigZ> {}

const reqZ = z.object({
  command: z.enum(WriterCommand),
  config: netConfigZ.optional(),
  frame: frameZ.optional(),
  buffer: z.instanceof(Uint8Array).optional(),
});

export interface WriteRequest extends z.infer<typeof reqZ> {}

const resZ = z.object({
  command: z.enum(WriterCommand),
  end: TimeStamp.z,
  err: errors.payloadZ.optional(),
});

interface Response extends z.infer<typeof resZ> {}

export interface WriterConfig {
  // channels denote the channels to write to.
  channels: channel.Params;
  // start sets the starting timestamp for the first sample in the writer.
  start?: CrudeTimeStamp;
  // controlSubject sets the control subject of the writer.
  controlSubject?: control.Subject;
  // authorities set the control authority to set for each channel on the writer.
  // Defaults to absolute authority. If not working with concurrent control,
  // it's best to leave this as the default.
  authorities?: control.Authority | control.Authority[];
  // mode sets the persistence and streaming mode of the writer. The default
  // mode is WriterModePersistStream.
  mode?: CrudeWriterMode;
  // errOnUnauthorized sets whether the writer raises an error when it attempts to write
  // to a channel without permission.
  errOnUnauthorized?: boolean;
  //  enableAutoCommit determines whether the writer will automatically commit.
  //  If enableAutoCommit is true, then the writer will commit after each write, and
  //  will flush that commit to index after the specified autoIndexPersistInterval.
  enableAutoCommit?: boolean;
  // autoIndexPersistInterval sets the interval at which commits to the index will be
  // persisted. To persist every commit to guarantee minimal loss of data, set
  // auto_index_persist_interval to AlwaysAutoIndexPersist.
  autoIndexPersistInterval?: TimeSpan;
  useExperimentalCodec?: boolean;
}

/**
 * Writer is used to write telemetry to a set of channels in time order.
 * It should not be instantiated directly, and should instead be instantiated via the
 * FramerClient {@link FrameClient#openWriter}.
 *
 * The writer is a streaming protocol that is heavily optimized for performance. This
 * comes at the cost of increased complexity, and should only be used directly when
 * writing large volumes of data (such as recording telemetry from a sensor or ingesting
 * data from file). Simpler methods (such as the frame client's write method) should
 * be used for most use cases.
 *
 * The protocol is as follows:
 *
 * 1. The writer is opened with a starting timestamp and a list of channel keys. The
 * writer will fail to open if the starting timestamp overlaps with any existing telemetry
 * for any channels specified. If the writer opens successfully, the caller is then
 * free to write frames to the writer.
 *
 * 2. To write a frame, the caller can use the write method and follow the validation
 * rules described in its method's documentation. This process is asynchronous, meaning
 * that write calls may return before the frame has been written to the cluster. This
 * also means that the writer can accumulate an error after write is called. If the writer
 * accumulates an error, all subsequent write and commit calls will return False. The
 * caller can check for errors by calling the error method, which returns the accumulated
 * error and resets the writer for future use. The caller can also check for errors by
 * closing the writer, which will throw any accumulated error.
 *
 * 3. To commit the written frames to the cluster, the caller can call the commit method.
 * Unlike write, commit is synchronous, meaning that it will not return until the frames
 * have been written to the cluster. If the writer has accumulated an error, commit will
 * return false. After the caller acknowledges the error, they can attempt to commit again.
 * Commit can be called several times throughout a writer's lifetime, and will only
 * commit the frames that have been written since the last commit.
 *
 * 4. A writer MUST be closed after use in order to prevent resource leaks. Close should
 * typically be called in a 'finally' block. If the writer has accumulated an error,
 * close will throw the error.
 */
export class Writer {
  private static readonly ENDPOINT = "/frame/write";
  private readonly stream: Stream<typeof reqZ, typeof resZ>;
  private readonly adapter: WriteAdapter;
  private closeErr: Error | null = null;

  private constructor(stream: Stream<typeof reqZ, typeof resZ>, adapter: WriteAdapter) {
    this.stream = stream;
    this.adapter = adapter;
  }

  static async _open(
    retriever: channel.Retriever,
    client: WebSocketClient,
    {
      channels,
      start = TimeStamp.now(),
      authorities = control.ABSOLUTE_AUTHORITY,
      controlSubject: subject,
      mode = WriterMode.PersistStream,
      errOnUnauthorized = false,
      enableAutoCommit = false,
      autoIndexPersistInterval = TimeSpan.SECOND,
      useExperimentalCodec = true,
    }: WriterConfig,
  ): Promise<Writer> {
    const adapter = await WriteAdapter.open(retriever, channels);
    if (useExperimentalCodec)
      client = client.withCodec(new WSWriterCodec(adapter.codec));
    const stream = await client.stream(Writer.ENDPOINT, reqZ, resZ);
    const writer = new Writer(stream, adapter);
    await writer.execute({
      command: WriterCommand.Open,
      config: {
        start: new TimeStamp(start),
        keys: adapter.keys,
        controlSubject: subject,
        authorities: array.toArray(authorities),
        mode: constructWriterMode(mode),
        errOnUnauthorized,
        enableAutoCommit,
        autoIndexPersistInterval,
      },
    });
    return writer;
  }

  async write(channel: channel.KeyOrName, data: CrudeSeries): Promise<void>;
  async write(channel: channel.KeysOrNames, data: CrudeSeries[]): Promise<void>;
  async write(
    frame: CrudeFrame | Record<channel.KeyOrName, CrudeSeries>,
  ): Promise<void>;
  async write(
    channelsOrData:
      | channel.Params
      | Record<channel.KeyOrName, CrudeSeries>
      | CrudeFrame,
    series?: CrudeSeries | CrudeSeries[],
  ): Promise<void>;

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
    channelsOrData:
      | channel.Params
      | Record<channel.KeyOrName, CrudeSeries>
      | CrudeFrame,
    series?: CrudeSeries | CrudeSeries[],
  ): Promise<void> {
    if (this.closeErr != null) throw this.closeErr;
    if (this.stream.received()) return await this.close();
    const frame = await this.adapter.adapt(channelsOrData, series);
    this.stream.send({ command: WriterCommand.Write, frame: frame.toPayload() });
  }

  async setAuthority(value: number): Promise<void>;

  async setAuthority(
    key: channel.KeyOrName,
    authority: control.Authority,
  ): Promise<void>;

  async setAuthority(
    value: Record<channel.KeyOrName, control.Authority>,
  ): Promise<void>;

  async setAuthority(
    value: Record<channel.KeyOrName, control.Authority> | channel.KeyOrName | number,
    authority?: control.Authority,
  ): Promise<void> {
    if (this.closeErr != null) throw this.closeErr;
    let config: Config;
    if (typeof value === "number" && authority == null)
      config = { keys: [], authorities: [value] };
    else {
      let oValue: Record<channel.KeyOrName, control.Authority>;
      if (typeof value === "string" || typeof value === "number")
        oValue = { [value]: authority } as Record<channel.KeyOrName, control.Authority>;
      else oValue = value;
      oValue = await this.adapter.adaptObjectKeys(oValue);
      config = {
        keys: Object.keys(oValue).map((k) => Number(k)),
        authorities: Object.values(oValue),
      };
    }
    await this.execute({ command: WriterCommand.SetAuthority, config });
  }

  /**
   * Commits the written frames to the database. Commit is synchronous, meaning that it
   * will not return until all frames have been committed to the database.
   *
   * @returns false if the commit failed due to an error. In this case, the caller
   * should acknowledge the error by calling the error method or closing the writer.
   * After the caller acknowledges the error, they can attempt to commit again.
   */
  async commit(): Promise<TimeStamp> {
    if (this.closeErr != null) throw this.closeErr;
    if (this.stream.received()) {
      await this.closeInternal(null);
      return TimeStamp.ZERO;
    }
    const res = await this.execute({ command: WriterCommand.Commit });
    return res.end;
  }

  /**
   * Closes the writer, raising any accumulated error encountered during operation.
   * A writer MUST be closed after use, and this method should probably be placed
   * in a 'finally' block.
   */
  async close(): Promise<void> {
    await this.closeInternal(null);
  }

  private async closeInternal(err: Error | null): Promise<null> {
    if (this.closeErr != null) throw this.closeErr;
    this.closeErr = err;
    this.stream.closeSend();
    while (true) {
      if (this.closeErr != null) {
        if (WriterClosedError.matches(this.closeErr)) return null;
        throw this.closeErr;
      }
      const [res, err] = await this.stream.receive();
      if (err != null) this.closeErr = EOF.matches(err) ? new WriterClosedError() : err;
      else this.closeErr = errors.decode(res?.err);
    }
  }

  private async execute(req: WriteRequest): Promise<Response> {
    const err = this.stream.send(req);
    if (err != null) await this.closeInternal(err);
    while (true) {
      const [res, err] = await this.stream.receive();
      if (err != null) await this.closeInternal(err);
      const resErr = errors.decode(res?.err);
      if (resErr != null) await this.closeInternal(resErr);
      if (res?.command == req.command) return res;
    }
  }
}
