// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type WebsocketMessage } from "@synnaxlabs/freighter";
import {
  binary,
  DataType,
  type SeriesPayload,
  TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";
import { type ZodSchema } from "zod";

import { type channel } from "@/channel";
import { type Payload } from "@/framer/frame";
import { type StreamerResponse } from "@/framer/streamer";
import { WriterCommand, type WriteRequest } from "@/framer/writer";

// For detailed information about the specifications,
// please refer to the official RFC 0016 document.
// Document here: docs/tech/rfc/0016-231001-frame-flight-protocol.md

const seriesPldLength = (series: SeriesPayload): number =>
  series.data.byteLength / series.dataType.density.valueOf();

interface KeyedSeries extends SeriesPayload {
  key: number;
}

const sortFramePayloadByKey = (framePayload: Payload): void => {
  const { keys, series } = framePayload;
  keys.forEach((key, index) => {
    (series[index] as KeyedSeries).key = key;
  });
  series.sort((a, b) => (a as KeyedSeries).key - (b as KeyedSeries).key);
  keys.sort((a, b) => a - b);
  // @ts-expect-error - deleting static property keys.
  series.forEach((ser) => delete (ser as KeyedSeries).key);
};

const ZERO_ALIGNMENTS_FLAG_POS = 5;
const EQUAL_ALIGNMENTS_FLAG_POS = 4;
const EQUAL_LENGTHS_FLAG_POS = 3;
const EQUAL_TIME_RANGES_FLAG_POS = 2;
const TIME_RANGES_ZERO_FLAG_POS = 1;
const ALL_CHANNELS_PRESENT_FLAG_POS = 0;

const TIMESTAMP_SIZE = DataType.TIMESTAMP.density.valueOf();
const ALIGNMENT_SIZE = 8;
const DATA_LENGTH_SIZE = 4;
const KEY_SIZE = 4;

export class Codec {
  contentType: string = "application/sy-framer";
  private keys: channel.Keys = [];
  private prevKeys: channel.Keys = [];
  private keyDataTypes: Map<channel.Key, DataType> = new Map();
  private hasVariableDataTypes: boolean = false;

  constructor(keys: channel.Keys, dataTypes: DataType[]) {
    this.update(keys, dataTypes);
  }

  update(keys: channel.Keys, dataTypes: DataType[]): void {
    this.prevKeys = [...this.keys];
    this.keys = keys;
    this.keyDataTypes = new Map();
    keys.forEach((k, i) => {
      const dt = dataTypes[i];
      this.keyDataTypes.set(k, dt);
      if (dt.isVariable) this.hasVariableDataTypes = true;
    });
    this.keys.sort();
  }

  encode(payload: unknown, startOffset: number = 0): Uint8Array {
    const src = payload as Payload;
    sortFramePayloadByKey(src);
    let currDataSize = -1;
    let startTime: TimeStamp | undefined;
    let endTime: TimeStamp | undefined;
    let currAlignment: bigint | undefined;
    let byteArraySize = startOffset + 1;
    let equalLengthsFlag = !this.hasVariableDataTypes;
    let equalTimeRangesFlag = true;
    let timeRangesZeroFlag = true;
    let channelFlag = true;
    let equalAlignmentsFlag = true;
    let zeroAlignmentsFlag = true;

    if (src.keys.length !== this.keys.length) {
      channelFlag = false;
      byteArraySize += src.keys.length * KEY_SIZE;
    }

    src.series.forEach((series) => {
      const pldLength = seriesPldLength(series);
      byteArraySize += series.data.byteLength;
      if (currDataSize === -1) {
        currDataSize = pldLength;
        startTime = series.timeRange?.start;
        endTime = series.timeRange?.end;
        currAlignment = BigInt(series.alignment ?? 0n);
        return;
      }
      if (currDataSize !== pldLength) equalLengthsFlag = false;
      if (
        startTime?.valueOf() !== series.timeRange?.start.valueOf() ||
        endTime?.valueOf() !== series.timeRange?.end.valueOf()
      )
        equalTimeRangesFlag = false;
      if (currAlignment !== BigInt(series.alignment ?? 0)) equalAlignmentsFlag = false;
    });

    timeRangesZeroFlag = equalTimeRangesFlag && startTime == null && endTime == null;

    zeroAlignmentsFlag =
      equalAlignmentsFlag && (currAlignment === undefined || currAlignment === 0n);

    if (equalLengthsFlag) byteArraySize += DATA_LENGTH_SIZE;
    else byteArraySize += src.keys.length * DATA_LENGTH_SIZE;

    if (!timeRangesZeroFlag)
      if (equalTimeRangesFlag) byteArraySize += TIMESTAMP_SIZE * 2;
      else byteArraySize += src.keys.length * TIMESTAMP_SIZE * 2;

    if (!zeroAlignmentsFlag)
      if (equalAlignmentsFlag) byteArraySize += ALIGNMENT_SIZE;
      else byteArraySize += src.keys.length * ALIGNMENT_SIZE;

    const buffer = new Uint8Array(byteArraySize);
    const view = new DataView(buffer.buffer);
    let offset = startOffset;
    buffer[offset] =
      (Number(zeroAlignmentsFlag) << ZERO_ALIGNMENTS_FLAG_POS) |
      (Number(equalAlignmentsFlag) << EQUAL_ALIGNMENTS_FLAG_POS) |
      (Number(equalLengthsFlag) << EQUAL_LENGTHS_FLAG_POS) |
      (Number(equalTimeRangesFlag) << EQUAL_TIME_RANGES_FLAG_POS) |
      (Number(timeRangesZeroFlag) << TIME_RANGES_ZERO_FLAG_POS) |
      (Number(channelFlag) << ALL_CHANNELS_PRESENT_FLAG_POS);
    offset++;

    if (equalLengthsFlag) {
      view.setUint32(offset, currDataSize, true);
      offset += DATA_LENGTH_SIZE;
    }

    if (equalTimeRangesFlag && !timeRangesZeroFlag) {
      view.setBigUint64(offset, startTime?.valueOf() ?? 0n, true);
      view.setBigUint64(offset, endTime?.valueOf() ?? 0n, true);
      offset += TIMESTAMP_SIZE * 2;
    }

    if (equalAlignmentsFlag && !zeroAlignmentsFlag) {
      view.setBigUint64(offset, currAlignment ?? 0n, true);
      offset += ALIGNMENT_SIZE;
    }

    src.series.forEach((series, i) => {
      if (!channelFlag) {
        view.setUint32(offset, src.keys[i], true);
        offset += KEY_SIZE;
      }
      if (!equalLengthsFlag) {
        let seriesLengthOrSize = series.data.byteLength;
        if (!series.dataType.isVariable) seriesLengthOrSize = seriesPldLength(series);
        view.setUint32(offset, seriesLengthOrSize, true);
        offset += DATA_LENGTH_SIZE;
      }
      buffer.set(new Uint8Array(series.data), offset);
      offset += series.data.byteLength;
      if (!equalTimeRangesFlag && !timeRangesZeroFlag) {
        view.setBigUint64(offset, series.timeRange?.start.valueOf() ?? 0n, true);
        view.setBigUint64(offset, series.timeRange?.end.valueOf() ?? 0n, true);
        offset += TIMESTAMP_SIZE * 2;
      }
      if (!equalAlignmentsFlag && !zeroAlignmentsFlag) {
        view.setBigUint64(offset, series.alignment ?? 0n, true);
        offset += ALIGNMENT_SIZE;
      }
    });
    return buffer;
  }

  decode(data: Uint8Array | ArrayBuffer, offset: number = 0): Payload {
    const src = data instanceof Uint8Array ? data : new Uint8Array(data);
    const returnFrame: Payload = { keys: [], series: [] };
    let index = offset;
    let sizeRepresentation = 0;
    let currSize = 0;
    let startTime: TimeStamp | undefined;
    let endTime: TimeStamp | undefined;
    let currAlignment: bigint | undefined;

    const view = new DataView(src.buffer, src.byteOffset, src.byteLength);
    const zeroAlignmentsFlag = Boolean((src[index] >> ZERO_ALIGNMENTS_FLAG_POS) & 1);
    const equalAlignmentsFlag = Boolean((src[index] >> EQUAL_ALIGNMENTS_FLAG_POS) & 1);
    const sizeFlag = Boolean((src[index] >> EQUAL_LENGTHS_FLAG_POS) & 1);
    const equalTimeRangesFlag = Boolean((src[index] >> EQUAL_TIME_RANGES_FLAG_POS) & 1);
    const timeRangesZeroFlag = Boolean((src[index] >> TIME_RANGES_ZERO_FLAG_POS) & 1);
    const channelFlag = Boolean((src[index] >> ALL_CHANNELS_PRESENT_FLAG_POS) & 1);
    index++;

    if (sizeFlag) {
      if (index + DATA_LENGTH_SIZE > view.byteLength) return returnFrame;
      sizeRepresentation = view.getUint32(index, true);
      index += DATA_LENGTH_SIZE;
    }

    if (equalTimeRangesFlag && !timeRangesZeroFlag) {
      if (index + TIMESTAMP_SIZE > view.byteLength) return returnFrame;
      startTime = new TimeStamp(view.getBigUint64(index, true));
      index += TIMESTAMP_SIZE;
      endTime = new TimeStamp(view.getBigUint64(index, true));
      index += TIMESTAMP_SIZE;
    }

    if (equalAlignmentsFlag && !zeroAlignmentsFlag) {
      if (index + ALIGNMENT_SIZE > view.byteLength) return returnFrame;
      currAlignment = view.getBigUint64(index, true);
      index += ALIGNMENT_SIZE;
    }

    if (channelFlag) returnFrame.keys = [...this.keys];
    this.keys.forEach((k, i) => {
      if (!channelFlag) {
        if (index + KEY_SIZE > view.byteLength) return;
        const frameKey = view.getUint32(index, true);
        if (frameKey !== k) return;
        index += KEY_SIZE;
        returnFrame.keys.push(k);
      }
      const dataType = this.keyDataTypes.get(k) as DataType;
      currSize = 0;
      if (!sizeFlag) {
        if (index + DATA_LENGTH_SIZE > view.byteLength) return;
        currSize = view.getUint32(index, true);
        index += DATA_LENGTH_SIZE;
      } else currSize = sizeRepresentation;

      let dataByteLength = currSize;
      if (!dataType.isVariable) dataByteLength *= dataType.density.valueOf();
      if (index + dataByteLength > view.byteLength) {
        returnFrame.keys.splice(i, 1);
        return;
      }
      const currSeries: SeriesPayload = {
        dataType,
        data: src.slice(index, index + dataByteLength).buffer,
      };
      index += dataByteLength;
      if (!equalTimeRangesFlag && !timeRangesZeroFlag) {
        if (index + TIMESTAMP_SIZE * 2 > view.byteLength) return;
        const start = view.getBigUint64(index, true);
        index += TIMESTAMP_SIZE;
        const end = view.getBigUint64(index, true);
        index += TIMESTAMP_SIZE;
        currSeries.timeRange = new TimeRange({ start, end });
      } else if (!timeRangesZeroFlag)
        currSeries.timeRange = new TimeRange({
          start: startTime?.valueOf() ?? 0n,
          end: endTime?.valueOf() ?? 0n,
        });
      else currSeries.timeRange = new TimeRange({ start: 0n, end: 0n });

      if (!equalAlignmentsFlag && !zeroAlignmentsFlag) {
        if (index + ALIGNMENT_SIZE > view.byteLength) return;
        currAlignment = view.getBigUint64(index, true);
        index += ALIGNMENT_SIZE;
        currSeries.alignment = currAlignment;
      } else if (!zeroAlignmentsFlag) currSeries.alignment = currAlignment;
      else currSeries.alignment = 0n;

      returnFrame.series.push(currSeries);
    });
    if (returnFrame.keys.length !== returnFrame.series.length)
      return { keys: [], series: [] };
    return returnFrame;
  }
}

export const LOW_PER_SPECIAL_CHAR = 254;
const LOW_PERF_SPECIAL_CHAR_BUF = new Uint8Array([LOW_PER_SPECIAL_CHAR]);
export const HIGH_PERF_SPECIAL_CHAR = 255;
const HIGH_PERF_SPECIAL_CHAR_BUF = new Uint8Array([HIGH_PERF_SPECIAL_CHAR]);
const CONTENT_TYPE = "application/sy-framer";

export class WSWriterCodec implements binary.Codec {
  contentType = CONTENT_TYPE;
  private base: Codec;
  private lowPerfCodec: binary.Codec;

  constructor(base: Codec) {
    this.base = base;
    this.lowPerfCodec = binary.JSON_CODEC;
  }

  encode(payload: unknown): Uint8Array {
    const pld = payload as WebsocketMessage<WriteRequest>;
    if (pld.type == "close" || pld.payload?.command != WriterCommand.Write) {
      const data = this.lowPerfCodec.encode(pld);
      const b = new Uint8Array({ length: data.byteLength + 1 });
      b.set(LOW_PERF_SPECIAL_CHAR_BUF, 0);
      b.set(new Uint8Array(data), 1);
      return b;
    }
    const data = this.base.encode(pld.payload?.frame, 1);
    data.set(HIGH_PERF_SPECIAL_CHAR_BUF, 0);
    return data;
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
  contentType = CONTENT_TYPE;
  private base: Codec;
  private lowPerfCodec: binary.Codec;

  constructor(base: Codec) {
    this.base = base;
    this.lowPerfCodec = binary.JSON_CODEC;
  }

  encode(payload: unknown): Uint8Array {
    return this.lowPerfCodec.encode(payload);
  }

  decode<P>(data: Uint8Array | ArrayBuffer, schema?: ZodSchema<P>): P {
    const dv = new DataView(data instanceof Uint8Array ? data.buffer : data);
    const codec = dv.getUint8(0);
    if (codec === LOW_PER_SPECIAL_CHAR)
      return this.lowPerfCodec.decode(data.slice(1), schema);
    const v: WebsocketMessage<StreamerResponse> = {
      type: "data",
      payload: { frame: this.base.decode(data, 1) },
    };
    return v as P;
  }
}
