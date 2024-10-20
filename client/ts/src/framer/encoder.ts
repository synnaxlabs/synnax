// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, DataType, SeriesPayload, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { ZodSchema } from "zod";

import { channel } from "@/channel";
import { FramePayload } from "@/framer/frame";

// For detailed information about the specifications,
// please refer to the official RFC 0016 document.
// Document here: docs/tech/rfc/0016-231001-frame-flight-protocol.md

const seriesPldLength = (series: SeriesPayload): number =>
  series.data.byteLength / series.dataType.density.valueOf();

export class Codec implements binary.Codec {
  contentType: string = "application/framer";
  private readonly keys: channel.Keys;
  private readonly dataTypes: DataType[];
  private readonly keyDataTypes: Map<channel.Key, DataType>;

  constructor(keys: channel.Keys, dataTypes: DataType[]) {
    this.keys = keys;
    this.dataTypes = dataTypes;
    this.keyDataTypes = new Map();
    keys.forEach((k, i) => this.keyDataTypes.set(k, dataTypes[i]));
  }

  encode(payload: unknown): Uint8Array {
    const src = payload as FramePayload;
    let currDataSize = -1;
    let startTime: TimeStamp | undefined = undefined;
    let endTime: TimeStamp | undefined = undefined;
    let byteArraySize = 1;
    let sizeFlag = true;
    let alignFlag = true;
    let channelFlag = true;

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
      }
      if (currDataSize !== pldLength) sizeFlag = false;
      if (startTime !== series.timeRange?.start || endTime !== series.timeRange?.end)
        alignFlag = false;
      byteArraySize += series.data.byteLength;
    });
    byteArraySize += (sizeFlag ? 1 : src.keys.length) * 4;
    byteArraySize += (alignFlag ? 1 : src.keys.length) * 16;
    const buffer = new Uint8Array(byteArraySize);
    const view = new DataView(buffer.buffer);
    byteArraySize = 0;
    buffer[byteArraySize] =
      (Number(sizeFlag) << 2) | (Number(alignFlag) << 1) | Number(channelFlag);
    byteArraySize++;

    if (sizeFlag) {
      view.setUint32(byteArraySize, currDataSize, true);
      byteArraySize += 4;
    }
    if (alignFlag) {
      view.setBigUint64(byteArraySize, BigInt(startTime ?? 0n), true);
      byteArraySize += 8;
      view.setBigUint64(byteArraySize, BigInt(endTime ?? 0n), true);
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
      if (!alignFlag) {
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
    });
    return buffer;
  }

  /**
   * Decodes the given binary representation into a type checked payload.
   *
   * @param data - The data to decode.
   * @param schema - The schema to decode the data with.
   */
  decode<P>(data: Uint8Array | ArrayBuffer, schema?: ZodSchema<P>): P {
    const src = data instanceof Uint8Array ? data : new Uint8Array(data);
    const returnFrame: FramePayload = { keys: [], series: [] };
    let index = 0;
    let sizeRepresentation = 0;
    let currSize = 0;
    let startTime = 0n;
    let endTime = 0n;

    const view = new DataView(src.buffer, src.byteOffset, src.byteLength);
    const sizeFlag = Boolean((src[index] >> 2) & 1);
    const alignFlag = Boolean((src[index] >> 1) & 1);
    const channelFlag = Boolean(src[index] & 1);
    index++;

    if (sizeFlag) {
      sizeRepresentation = view.getUint32(index, true);
      index += 4;
    }
    if (alignFlag) {
      startTime = view.getBigUint64(index, true);
      index += 8;
      endTime = view.getBigUint64(index, true);
      index += 8;
    }

    if (channelFlag) returnFrame.keys = this.keys;
    this.keys.forEach((k) => {
      if (!channelFlag) {
        if (index >= src.length) return;
        const ok = view.getUint32(index, true);
        if (ok !== k) return;
        returnFrame.keys.push(k);
        index += 4;
      }
      const dataType = this.keyDataTypes.get(k) as DataType;
      currSize = 0;
      if (!sizeFlag) {
        currSize = view.getUint32(index, true);
        index += 4;
      } else {
        currSize = sizeRepresentation;
      }
      const currSeries: SeriesPayload = {
        dataType,
        data: src.slice(index, index + currSize * dataType.density.valueOf()).buffer,
      };
      index += currSize * dataType.density.valueOf();
      if (!alignFlag) {
        const start = BigInt(view.getBigUint64(index, true));
        index += 16;
        const end = BigInt(view.getBigUint64(index, true));
        index += 16;
        currSeries.timeRange = new TimeRange({ start, end });
      } else {
        currSeries.timeRange = new TimeRange({
          start: BigInt(startTime),
          end: BigInt(endTime),
        });
      }
      returnFrame.series.push(currSeries);
    });
    return returnFrame as P;
  }
}
