// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream, type WebSocketClient } from "@synnaxlabs/freighter";
import {
  control,
  type CrudeSeries,
  errors,
  TimeSpan,
  TimeStamp,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import { channel } from "@/channel";
import { SynnaxError } from "@/errors";
import { WriteAdapter } from "@/framer/adapter";
import { WSWriterCodec } from "@/framer/codec";
import { type CrudeFrame, frameZ } from "@/framer/frame";
import { WriterCommand } from "@/framer/types.gen";

export enum WriterMode {
  PersistStream = 1,
  Persist = 2,
  Stream = 3,
}

export const ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT: TimeSpan = new TimeSpan(-1);

export class WriterClosedError extends SynnaxError.sub("writer_closed") {
  constructor() {
    super("WriterClosed");
  }
}

const writerModeZ = z.enum(WriterMode).or(
  z.enum(["persist", "stream", "persistStream"]).transform((mode) => {
    switch (mode) {
      case "persist":
        return WriterMode.Persist;
      case "stream":
        return WriterMode.Stream;
      case "persistStream":
        return WriterMode.PersistStream;
    }
  }),
);

export type CrudeWriterMode = z.input<typeof writerModeZ>;

const baseWriterConfigZ = z.object({
  /** start sets the starting timestamp for the first sample in the writer. */
  start: TimeStamp.z.optional(),
  /** controlSubject sets the control subject of the writer. */
  controlSubject: control.subjectZ.optional(),
  /** authorities set the control authority to set for each channel on the writer.
   * Defaults to absolute authority. If not working with concurrent control, it's best
   * to leave this as the default.
   */
  authorities: z
    .union([control.authorityZ.transform((a) => [a]), control.authorityZ.array()])
    .default([control.ABSOLUTE_AUTHORITY]),
  /** mode sets the persistence and streaming mode of the writer. The default mode is
   * WriterModePersistStream.
   */
  mode: writerModeZ.default(WriterMode.PersistStream),
  /**
   * errOnUnauthorized sets whether the writer raises an error when it attempts to write
   * to a channel without permission.
   */
  errOnUnauthorized: z.boolean().default(false),
  /**
   * enableAutoCommit determines whether the writer will automatically commit. If
   * enableAutoCommit is true, then the writer will commit after each write, and will
   * flush that commit to index after the specified autoIndexPersistInterval.
   */
  enableAutoCommit: z.boolean().default(true),
  /** autoIndexPersistInterval sets the interval at which commits will be flushed to
   * disk. */
  autoIndexPersistInterval: TimeSpan.z.default(TimeSpan.SECOND),
  /**
   * autoIndex causes Synnax to automatically generate timestamps for any index channel
   * that is not included in a write call. The first sample in each write is stamped at
   * the time the write is received, and subsequent samples are spaced 1 nanosecond
   * apart. Generated timestamps are guaranteed to be strictly monotonic across all
   * writes on the writer. If the writer is opened with data channels whose index
   * channels are not included, those index channels are added implicitly.
   */
  autoIndex: z.boolean().default(false),
});

const netWriterConfigZ = baseWriterConfigZ.extend({
  keys: channel.keyZ.array().optional(),
});

export type NetWriterConfig = z.input<typeof netWriterConfigZ>;

// Intermediate utility type to allow for the use of paramsZ in the writer config.
const intermediateWriterConfigZ = baseWriterConfigZ.extend({
  channels: channel.paramsZ,
});

export const writerConfigZ = intermediateWriterConfigZ.or(
  channel.paramsZ.transform((channels) =>
    intermediateWriterConfigZ.parse({ channels, start: TimeStamp.now() }),
  ),
);

export type WriterConfig = z.input<typeof writerConfigZ>;

const reqZ = z.object({
  command: z.enum(WriterCommand),
  config: netWriterConfigZ.optional(),
  frame: frameZ.optional(),
  buffer: z.instanceof(Uint8Array).optional(),
});

export interface WriteRequest extends z.input<typeof reqZ> {}

const resZ = z.object({
  command: z.enum(WriterCommand),
  end: TimeStamp.z,
  err: errors.payloadZ.optional(),
});

const authorityArgsZ = z
  .tuple([
    z.union([
      z.record(channel.keyZ.or(channel.nameZ), control.authorityZ),
      channel.keyZ.or(channel.nameZ),
      control.authorityZ,
    ]),
    control.authorityZ.optional(),
  ])
  .transform(([value, authority]) => {
    if (control.authorityZ.safeParse(value).success)
      return { keys: [], authorities: [value as control.Authority] };
    if (channel.keyZ.or(channel.nameZ).safeParse(value).success) {
      if (authority == null)
        throw new Error(
          "authority is required when setting authority for a single channel",
        );
      return {
        keys: [value] as channel.Key[] | channel.Name[],
        authorities: [authority],
      };
    }
    const oValue = value as Record<channel.Key | channel.Name, control.Authority>;
    return { keys: Object.keys(oValue), authorities: Object.values(oValue) };
  });

export type AuthorityArgs = z.input<typeof authorityArgsZ>;

interface Response extends z.infer<typeof resZ> {}

/**
 * Writer is used to write telemetry to a set of channels in time order. It should not
 * be instantiated directly, and should instead be instantiated via the FramerClient
 * {@link FrameClient#openWriter}.
 *
 * The writer is a streaming protocol that is heavily optimized for performance. This
 * comes at the cost of increased complexity, and should only be used directly when
 * writing large volumes of data (such as recording telemetry from a sensor or ingesting
 * data from file). Simpler methods (such as the frame client's write method) should be
 * used for most use cases.
 *
 * The protocol is as follows:
 *
 * 1. The writer is opened with a starting timestamp and a list of channel keys. The
 *    writer will fail to open if the starting timestamp overlaps with any existing
 *    telemetry for any channels specified. If the writer opens successfully, the caller
 *    is then free to write frames to the writer.
 *
 * 2. To write a frame, the caller can use the write method and follow the validation
 *    rules described in its method's documentation. This process is asynchronous,
 *    meaning that write calls may return before the frame has been written to the
 *    cluster. This also means that the writer can accumulate an error after write is
 *    called. If the writer accumulates an error, all subsequent write and commit calls
 *    will return False. The caller can check for errors by calling the error method,
 *    which returns the accumulated error and resets the writer for future use. The
 *    caller can also check for errors by closing the writer, which will throw any
 *    accumulated error.
 *
 * 3. To commit the written frames to the cluster, the caller can call the commit
 *    method. Unlike write, commit is synchronous, meaning that it will not return until
 *    the frames have been written to the cluster. If the writer has accumulated an
 *    error, commit will return false. After the caller acknowledges the error, they can
 *    attempt to commit again. Commit can be called several times throughout a writer's
 *    lifetime, and will only commit the frames that have been written since the last
 *    commit.
 *
 * 4. A writer MUST be closed after use in order to prevent resource leaks. Close should
 *    typically be called in a 'finally' block. If the writer has accumulated an error,
 *    close will throw the error.
 */
export class Writer {
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
    config: WriterConfig,
  ): Promise<Writer> {
    const cfg = zod.parse(writerConfigZ, config);
    const adapter = await WriteAdapter.open(retriever, cfg.channels);
    client = client.withCodec(new WSWriterCodec(adapter.codec));
    const stream = await client.stream("/frame/write", reqZ, resZ);
    const writer = new Writer(stream, adapter);
    await writer.execute({
      command: WriterCommand.Open,
      config: { ...cfg, keys: adapter.keys },
    });
    return writer;
  }

  async write(channel: channel.Key | channel.Name, data: CrudeSeries): Promise<void>;
  async write(
    channel: channel.Key[] | channel.Name[],
    data: CrudeSeries[],
  ): Promise<void>;
  async write(
    frame: CrudeFrame | Record<channel.Key | channel.Name, CrudeSeries>,
  ): Promise<void>;
  async write(
    channelsOrData:
      | channel.Params
      | Record<channel.Key | channel.Name, CrudeSeries>
      | CrudeFrame,
    series?: CrudeSeries | CrudeSeries[],
  ): Promise<void>;

  /**
   * Writes the given frame to the database.
   *
   * @param frame - The frame to write to the database. The frame must:
   *
   *    1. Have exactly one array for each key in the list of keys provided to the
   *       writer's open method.
   *    2. Have equal length arrays for each key.
   *    3. When writing to an index (i.e. TimeStamp) channel, the values must be
   *       monotonically increasing.
   *
   * @throws if the writer has accumulated an error. Once write throws, all subsequent
   * calls to write and commit will also throw, and the writer must be closed and
   * re-opened to continue writing.
   */
  async write(
    channelsOrData:
      | channel.Params
      | Record<channel.Key | channel.Name, CrudeSeries>
      | CrudeFrame,
    series?: CrudeSeries | CrudeSeries[],
  ): Promise<void> {
    if (this.closeErr != null) throw this.closeErr;
    if (this.stream.received()) return await this.close();
    const frame = await this.adapter.adapt(channelsOrData, series);
    try {
      this.stream.send({ command: WriterCommand.Write, frame: frame.toPayload() });
    } catch (err) {
      if (!EOF.matches(err)) throw errors.toError(err);
    }
  }

  async setAuthority(
    value: AuthorityArgs[0],
    authority?: AuthorityArgs[1],
  ): Promise<void> {
    if (this.closeErr != null) throw this.closeErr;
    const parsed = authorityArgsZ.parse([value, authority]);
    const config = {
      keys: await this.adapter.adaptParams(parsed.keys),
      authorities: parsed.authorities,
    };
    await this.execute({ command: WriterCommand.SetAuthority, config });
  }

  /**
   * Commits the written frames to the database. Commit is synchronous, meaning that it
   * will not return until all frames have been committed to the database.
   *
   * @returns the timestamp of the last sample written to the writer.
   * @throws if the commit fails or any previous writer method has thrown. Once commit
   * throws, the writer must be closed and re-opened to continue use.
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
      try {
        const res = await this.stream.receive();
        this.closeErr = errors.decode(res?.err);
      } catch (err) {
        const e = errors.toError(err);
        this.closeErr = EOF.matches(e) ? new WriterClosedError() : e;
      }
    }
  }

  private async execute(req: WriteRequest): Promise<Response> {
    try {
      this.stream.send(req);
    } catch (err) {
      // A send failure is always EOF or StreamClosed, never WriterClosedError, so
      // closeInternal re-throws here and the receive loop below is reached only when
      // the send succeeds.
      await this.closeInternal(errors.toError(err));
    }
    while (true) {
      let res: Response;
      try {
        res = await this.stream.receive();
      } catch (err) {
        await this.closeInternal(errors.toError(err));
        continue;
      }
      const resErr = errors.decode(res?.err);
      if (resErr != null) await this.closeInternal(resErr);
      if (res?.command == req.command) return res;
    }
  }
}
