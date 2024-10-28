// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { WebsocketMessage } from "@synnaxlabs/freighter";
import { binary, DataType, SeriesPayload, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { ZodSchema } from "zod";

import { channel } from "@/channel";
import { FramePayload } from "@/framer/frame";
import { StreamerResponse } from "@/framer/streamer";
import { WriterCommand, WriteRequest } from "@/framer/writer";

// For detailed information about the specifications,
// please refer to the official RFC 0016 document.
// Document here: docs/tech/rfc/0016-231001-frame-flight-protocol.md

const seriesPldLength = (series: SeriesPayload): number =>
  series.data.byteLength / series.dataType.density.valueOf();

interface KeyedSeries extends SeriesPayload {
  key: number;
}

const sortFramePayloadByKey = (framePayload: FramePayload): void => {
  const { keys, series } = framePayload;
  keys.forEach((key, index) => {
    (series[index] as KeyedSeries).key = key;
  });
  series.sort((a, b) => (a as KeyedSeries).key - (b as KeyedSeries).key);
  keys.sort((a, b) => a - b);
  // @ts-expect-error - deleting static property keys.
  series.forEach((ser) => delete (ser as KeyedSeries).key);
};

// Adjusted TypeScript Codec Class

// Constants for flag positions
const zeroAlignmentsFlagPos = 5;
const equalAlignmentsFlagPos = 4;
const equalLengthsFlagPos = 3;
const equalTimeRangesFlagPos = 2;
const timeRangesZeroFlagPos = 1;
const allChannelsPresentFlagPos = 0;

export class Codec {
  contentType: string = "application/sy-framer";
  private readonly keys: channel.Keys;
  private readonly keyDataTypes: Map<channel.Key, DataType>;

  constructor(keys: channel.Keys, dataTypes: DataType[]) {
    this.keys = keys;
    this.keyDataTypes = new Map();
    keys.forEach((k, i) => this.keyDataTypes.set(k, dataTypes[i]));
    this.keys.sort();
  }

  encode(payload: unknown, startOffset: number = 0): Uint8Array {
    const src = payload as FramePayload;
    sortFramePayloadByKey(src);
    let currDataSize = -1;
    let startTime: TimeStamp | undefined = undefined;
    let endTime: TimeStamp | undefined = undefined;
    let currAlignment: bigint | undefined = undefined;
    let byteArraySize = startOffset + 1;
    let sizeFlag = true;
    let equalTimeRangesFlag = true;
    let timeRangesZeroFlag = true;
    let channelFlag = true;
    let equalAlignmentsFlag = true;
    let zeroAlignmentsFlag = true;

    if (src.keys.length !== this.keys.length) {
      channelFlag = false;
      byteArraySize += src.keys.length * 4;
    }

    src.series.forEach((series) => {
      const pldLength = seriesPldLength(series);
      if (currDataSize === -1) {
        currDataSize = pldLength;
        startTime = series.timeRange?.start;
        endTime = series.timeRange?.end;
        currAlignment = BigInt(series.alignment ?? 0n);
      }
      if (currDataSize !== pldLength) sizeFlag = false;
      if (
        startTime?.valueOf() !== series.timeRange?.start.valueOf() ||
        endTime?.valueOf() !== series.timeRange?.end.valueOf()
      )
        equalTimeRangesFlag = false;
      if (currAlignment !== BigInt(series.alignment ?? 0)) equalAlignmentsFlag = false;
      byteArraySize += series.data.byteLength;
    });

    timeRangesZeroFlag = equalTimeRangesFlag && startTime == null && endTime == null;

    zeroAlignmentsFlag =
      equalAlignmentsFlag && (currAlignment === undefined || currAlignment === 0n);

    if (sizeFlag) byteArraySize += 4;
    else byteArraySize += src.keys.length * 4;

    if (!timeRangesZeroFlag)
      if (equalTimeRangesFlag) byteArraySize += 16;
      else byteArraySize += src.keys.length * 16;

    if (!zeroAlignmentsFlag)
      if (equalAlignmentsFlag) byteArraySize += 8;
      else byteArraySize += src.keys.length * 8;

    const buffer = new Uint8Array(byteArraySize);
    const view = new DataView(buffer.buffer);
    byteArraySize = startOffset;
    buffer[byteArraySize] =
      (Number(zeroAlignmentsFlag) << zeroAlignmentsFlagPos) |
      (Number(equalAlignmentsFlag) << equalAlignmentsFlagPos) |
      (Number(sizeFlag) << equalLengthsFlagPos) |
      (Number(equalTimeRangesFlag) << equalTimeRangesFlagPos) |
      (Number(timeRangesZeroFlag) << timeRangesZeroFlagPos) |
      (Number(channelFlag) << allChannelsPresentFlagPos);
    byteArraySize++;

    if (sizeFlag) {
      view.setUint32(byteArraySize, currDataSize, true);
      byteArraySize += 4;
    }

    if (equalTimeRangesFlag && !timeRangesZeroFlag) {
      view.setBigUint64(byteArraySize, BigInt(startTime ?? 0n), true);
      byteArraySize += 8;
      view.setBigUint64(byteArraySize, BigInt(endTime ?? 0n), true);
      byteArraySize += 8;
    }

    if (equalAlignmentsFlag && !zeroAlignmentsFlag) {
      view.setBigUint64(byteArraySize, BigInt(currAlignment ?? 0n), true);
      byteArraySize += 8;
    }

    src.series.forEach((series, i) => {
      if (!channelFlag) {
        view.setUint32(byteArraySize, src.keys[i], true);
        byteArraySize += 4;
      }
      if (!sizeFlag) {
        const seriesLength = seriesPldLength(series);
        view.setUint32(byteArraySize, seriesLength, true);
        byteArraySize += 4;
      }
      buffer.set(new Uint8Array(series.data), byteArraySize);
      byteArraySize += series.data.byteLength;
      if (!equalTimeRangesFlag && !timeRangesZeroFlag) {
        view.setBigUint64(
          byteArraySize,
          BigInt(series.timeRange?.start.valueOf() ?? 0n),
          true,
        );
        byteArraySize += 8;
        view.setBigUint64(
          byteArraySize,
          BigInt(series.timeRange?.end.valueOf() ?? 0n),
          true,
        );
        byteArraySize += 8;
      }
      if (!equalAlignmentsFlag && !zeroAlignmentsFlag) {
        view.setBigUint64(byteArraySize, BigInt(series.alignment ?? 0n), true);
        byteArraySize += 8;
      }
    });
    return buffer;
  }

  decode(data: Uint8Array | ArrayBuffer, offset: number = 0): FramePayload {
    const src = data instanceof Uint8Array ? data : new Uint8Array(data);
    const returnFrame: FramePayload = { keys: [], series: [] };
    let index = offset;
    let sizeRepresentation = 0;
    let currSize = 0;
    let startTime = 0n;
    let endTime = 0n;
    let currAlignment = 0n;

    const view = new DataView(src.buffer, src.byteOffset, src.byteLength);
    const zeroAlignmentsFlag = Boolean((src[index] >> zeroAlignmentsFlagPos) & 1);
    const equalAlignmentsFlag = Boolean((src[index] >> equalAlignmentsFlagPos) & 1);
    const sizeFlag = Boolean((src[index] >> equalLengthsFlagPos) & 1);
    const equalTimeRangesFlag = Boolean((src[index] >> equalTimeRangesFlagPos) & 1);
    const timeRangesZeroFlag = Boolean((src[index] >> timeRangesZeroFlagPos) & 1);
    const channelFlag = Boolean((src[index] >> allChannelsPresentFlagPos) & 1);
    index++;

    if (sizeFlag) {
      if (index + 4 > view.byteLength) return returnFrame;
      sizeRepresentation = view.getUint32(index, true);
      index += 4;
    }

    if (equalTimeRangesFlag && !timeRangesZeroFlag) {
      if (index + 16 > view.byteLength) return returnFrame;
      startTime = view.getBigUint64(index, true);
      index += 8;
      endTime = view.getBigUint64(index, true);
      index += 8;
    }

    if (equalAlignmentsFlag && !zeroAlignmentsFlag) {
      if (index + 8 > view.byteLength) return returnFrame;
      currAlignment = view.getBigUint64(index, true);
      index += 8;
    }

    if (channelFlag) returnFrame.keys = this.keys;
    this.keys.forEach((k) => {
      if (!channelFlag) {
        if (index + 4 > view.byteLength) return;
        const ok = view.getUint32(index, true);
        index += 4;
        if (ok !== k) return;
        returnFrame.keys.push(k);
      }
      const dataType = this.keyDataTypes.get(k) as DataType;
      currSize = 0;
      if (!sizeFlag) {
        if (index + 4 > view.byteLength) return;
        currSize = view.getUint32(index, true);
        index += 4;
      } else {
        currSize = sizeRepresentation;
      }
      const dataByteLength = currSize * dataType.density.valueOf();
      if (index + dataByteLength > view.byteLength) return;
      const currSeries: SeriesPayload = {
        dataType,
        data: src.slice(index, index + dataByteLength).buffer,
      };
      index += dataByteLength;
      if (!equalTimeRangesFlag && !timeRangesZeroFlag) {
        if (index + 16 > view.byteLength) return;
        const start = view.getBigUint64(index, true);
        index += 8;
        const end = view.getBigUint64(index, true);
        index += 8;
        currSeries.timeRange = new TimeRange({ start, end });
      } else if (!timeRangesZeroFlag) {
        currSeries.timeRange = new TimeRange({
          start: BigInt(startTime),
          end: BigInt(endTime),
        });
      } else {
        currSeries.timeRange = new TimeRange({ start: 0n, end: 0n });
      }
      if (!equalAlignmentsFlag && !zeroAlignmentsFlag) {
        if (index + 8 > view.byteLength) return;
        currAlignment = view.getBigUint64(index, true);
        index += 8;
        currSeries.alignment = currAlignment;
      } else if (!zeroAlignmentsFlag) {
        currSeries.alignment = currAlignment;
      } else {
        currSeries.alignment = 0n;
      }
      returnFrame.series.push(currSeries);
    });
    return returnFrame;
  }
}

export const LOW_PER_SPECIAL_CHAR = 254;
const LOW_PERF_SPECIAL_CHAR_BUF = new Uint8Array([LOW_PER_SPECIAL_CHAR]);
export const HIGH_PERF_SPECIAL_CHAR = 255;
const HIGH_PERF_SPECIAL_CHAR_BUF = new Uint8Array([HIGH_PERF_SPECIAL_CHAR]);

export class WSWriterCodec implements binary.Codec {
  contentType: string = "application/sy-framer";
  base: Codec;
  lowPerfCodec: binary.Codec;

  constructor(base: Codec) {
    this.base = base;
    this.lowPerfCodec = binary.JSON_CODEC;
  }

  encode(payload: unknown): ArrayBuffer {
    const pld = payload as WebsocketMessage<WriteRequest>;
    if (pld.type == "close" || pld.payload?.command != WriterCommand.Write) {
      const data = this.lowPerfCodec.encode(pld);
      const b = new Uint8Array({ length: data.byteLength + 1 });
      b.set(LOW_PERF_SPECIAL_CHAR_BUF, 0);
      b.set(new Uint8Array(data), 1);
      return b.buffer;
    }
    const data = this.base.encode(pld.payload?.frame, 1);
    data.set(HIGH_PERF_SPECIAL_CHAR_BUF, 0);
    return data.buffer;
  }

  decode<P>(data: Uint8Array | ArrayBuffer, schema?: ZodSchema<P>): P {
    const dv = new DataView(data instanceof Uint8Array ? data.buffer : data);
    const codec = dv.getUint8(0);
    if (codec === LOW_PER_SPECIAL_CHAR)
      return this.lowPerfCodec.decode(data.slice(1), schema);
    const v: WebsocketMessage<WriteRequest> = { type: "data" };
    const frame = this.base.decode(data, 1);
    v.payload = { command: WriterCommand.Write, frame };
    return v as P;
  }
}

export class WSStreamerCodec implements binary.Codec {
  contentType = "application/sy-framer";
  base: Codec;
  lowPerfCodec: binary.Codec;

  constructor(base: Codec) {
    this.base = base;
    this.lowPerfCodec = binary.JSON_CODEC;
  }

  encode(payload: unknown): ArrayBuffer {
    console.log("ECD");
    return this.lowPerfCodec.encode(payload);
  }

  decode<P>(data: Uint8Array | ArrayBuffer, schema?: ZodSchema<P>): P {
    const dv = new DataView(data instanceof Uint8Array ? data.buffer : data);
    const codec = dv.getUint8(0);
    if (codec === LOW_PER_SPECIAL_CHAR)
      return this.lowPerfCodec.decode(data.slice(1), schema);
    const v: WebsocketMessage<StreamerResponse> = { type: "data" };
    const frame = this.base.decode(data, 1);
    v.payload = { frame };
    return v as P;
  }
}
