// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FileEncoding, type FileTransport } from "@synnaxlabs/freighter";
import { type CrudeTimeRange, errors, TimeRange } from "@synnaxlabs/x";
import { z } from "zod";

import { type channel } from "@/channel";
import { keyZ } from "@/channel/types.gen";
import { UnexpectedError } from "@/errors";
import { type ChannelRetriever, ReadAdapter } from "@/framer/adapter";
import { Frame } from "@/framer/frame";

const READ_ENDPOINT = "/frame/read";

const reqZ = z.object({
  keys: keyZ.array(),
  bounds: TimeRange.z,
  downsampleFactor: z.int(),
});

/** Options controlling how the Core resolves a read. */
export interface ReadOptions {
  /**
   * Keeps one sample in every downsampleFactor. A factor of 1, the default, keeps
   * every sample.
   */
  downsampleFactor?: number;
}

/**
 * Reader pulls historical telemetry from the Core in a single request per read, letting
 * the Core encode the response in whichever format the caller asked for.
 */
export class Reader {
  private readonly retrieveChannels: ChannelRetriever;
  private readonly file: FileTransport;

  constructor(retrieveChannels: ChannelRetriever, file: FileTransport) {
    this.retrieveChannels = retrieveChannels;
    this.file = file;
  }

  /**
   * Reads every sample the given channels hold within tr.
   *
   * @param tr - the time range to read.
   * @param channels - the channels to read, by key or by name. A frame read by name is
   * keyed by name.
   * @param opts - see {@link ReadOptions}.
   * @returns the samples as a frame.
   */
  async read(
    tr: CrudeTimeRange,
    channels: channel.Params,
    opts: ReadOptions = {},
  ): Promise<Frame> {
    const adapter = await ReadAdapter.open(this.retrieveChannels, channels);
    const stream = await this.download(Array.from(adapter.keys), tr, "FRAME", opts);
    const frame = new Frame();
    for await (const record of readRecords(stream))
      frame.push(new Frame(adapter.codec.decode(record)));
    return adapter.adapt(frame);
  }

  /**
   * Reads every sample the given channels hold within tr as CSV text. Channels are
   * grouped by the index channel that timestamps them, and each group contributes its
   * index column followed by one column per channel. Channels with no index are
   * excluded.
   *
   * @param tr - the time range to read.
   * @param channels - the channels to read, by key or by name.
   * @param opts - see {@link ReadOptions}.
   * @returns the CSV as a stream of bytes the caller pipes wherever it likes.
   */
  async readCSV(
    tr: CrudeTimeRange,
    channels: channel.Params,
    opts: ReadOptions = {},
  ): Promise<ReadableStream<Uint8Array>> {
    const payloads = await this.retrieveChannels(channels);
    return await this.download(
      payloads.map((c) => c.key),
      tr,
      "CSV",
      opts,
    );
  }

  private async download(
    keys: channel.Key[],
    tr: CrudeTimeRange,
    encoding: FileEncoding,
    opts: ReadOptions,
  ): Promise<ReadableStream<Uint8Array>> {
    return await this.file.download(
      READ_ENDPOINT,
      {
        keys,
        bounds: new TimeRange(tr),
        downsampleFactor: opts.downsampleFactor ?? 1,
      },
      reqZ,
      { encoding },
    );
  }
}

const HEADER_SIZE = 5;
const TERMINATOR_KIND = 1;

/**
 * Yields the payload of every data record in a frame-encoded read body. The terminator
 * ends the iteration, throwing the error it carries when the read failed partway. A
 * body that ends without one was cut short in transit.
 */
async function* readRecords(
  stream: ReadableStream<Uint8Array>,
): AsyncGenerator<Uint8Array> {
  const reader = stream.getReader();
  const chunks: Uint8Array[] = [];
  let available = 0;
  let offset = 0;
  let exhausted = false;
  // fill reads until the staged chunks hold size bytes, reporting false when the body
  // ends first.
  const fill = async (size: number): Promise<boolean> => {
    while (available < size && !exhausted) {
      const { value, done } = await reader.read();
      if (done) exhausted = true;
      else {
        chunks.push(value);
        available += value.length;
      }
    }
    return available >= size;
  };
  // take consumes the next size bytes, splicing across chunks only when one does not
  // cover them.
  const take = (size: number): Uint8Array => {
    if (size === 0) return new Uint8Array(0);
    const first = chunks[0];
    if (first.length - offset >= size) {
      const out = first.subarray(offset, offset + size);
      offset += size;
      available -= size;
      if (offset === first.length) {
        chunks.shift();
        offset = 0;
      }
      return out;
    }
    const out = new Uint8Array(size);
    let filled = 0;
    while (filled < size) {
      const chunk = chunks[0];
      const n = Math.min(size - filled, chunk.length - offset);
      out.set(chunk.subarray(offset, offset + n), filled);
      filled += n;
      offset += n;
      available -= n;
      if (offset === chunk.length) {
        chunks.shift();
        offset = 0;
      }
    }
    return out;
  };
  try {
    while (true) {
      if (!(await fill(HEADER_SIZE))) throw new UnexpectedError(TRUNCATED);
      const header = take(HEADER_SIZE);
      const kind = header[0];
      const size = new DataView(
        header.buffer,
        header.byteOffset,
        header.byteLength,
      ).getUint32(1, true);
      if (!(await fill(size))) throw new UnexpectedError(TRUNCATED);
      const payload = take(size);
      if (kind !== TERMINATOR_KIND) {
        yield payload;
        continue;
      }
      if (size === 0) return;
      throw (
        errors.decode(
          errors.payloadZ.parse(JSON.parse(new TextDecoder().decode(payload))),
        ) ?? new UnexpectedError(TRUNCATED)
      );
    }
  } finally {
    reader.releaseLock();
  }
}

const TRUNCATED = "The Core cut the read short before it finished.";
