// Copyright 2026 Synnax Labs, Inc.
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
  DataTypeCode,
  type SeriesPayload,
  TimeRange,
  type zod,
} from "@synnaxlabs/x";
import { type z } from "zod";

import { type channel } from "@/channel";
import { ValidationError } from "@/errors";
import { type Frame, type Payload } from "@/framer/frame";
import { type StreamerResponse } from "@/framer/streamer";
import { IteratorResponseVariant, WriterCommand } from "@/framer/types.gen";
import { type WriteRequest } from "@/framer/writer";

const BITS_PER_BYTE = 8;

const bitPackedByteCount = (nSamples: number): number =>
  Math.ceil(nSamples / BITS_PER_BYTE);

const seriesPldLength = (series: SeriesPayload): number =>
  series.data.byteLength / series.dataType.density.valueOf();

const seriesWireByteLength = (series: SeriesPayload): number => {
  if (series.dataType.equals(DataType.BOOLEAN))
    return bitPackedByteCount(seriesPldLength(series));
  return series.data.byteLength;
};

const packBoolBits = (src: ArrayBuffer | Uint8Array): Uint8Array => {
  const srcBytes = src instanceof Uint8Array ? src : new Uint8Array(src);
  const dst = new Uint8Array(bitPackedByteCount(srcBytes.length));
  for (let i = 0; i < srcBytes.length; i++)
    if (srcBytes[i] !== 0)
      dst[Math.floor(i / BITS_PER_BYTE)] |= 1 << (i % BITS_PER_BYTE);
  return dst;
};

const unpackBoolBits = (src: Uint8Array, sampleCount: number): ArrayBuffer => {
  const buf = new ArrayBuffer(sampleCount);
  const dst = new Uint8Array(buf);
  for (let i = 0; i < sampleCount; i++)
    dst[i] = (src[Math.floor(i / BITS_PER_BYTE)] >> (i % BITS_PER_BYTE)) & 1;
  return buf;
};

/** Orders channel keys the way the Core does. The default comparator sorts numbers as
 * strings and diverges as soon as two keys differ in decimal width. */
const compareKeys = (a: channel.Key, b: channel.Key): number => a - b;

const EXTENDED_FLAG_POS = 7;
const MULTIPLES_PRESENT_FLAG_POS = 6;
const ZERO_ALIGNMENTS_FLAG_POS = 5;
const EQUAL_ALIGNMENTS_FLAG_POS = 4;
const EQUAL_LENGTHS_FLAG_POS = 3;
const EQUAL_TIME_RANGES_FLAG_POS = 2;
const TIME_RANGES_ZERO_FLAG_POS = 1;
const ALL_CHANNELS_PRESENT_FLAG_POS = 0;

const TYPES_DECLARED = 1 << 0;
const OVERRIDES_PRESENT = 1 << 1;
const CLOCKS_DECLARED = 1 << 3;
// The extension bits and override kinds that no codec sets yet.
const RESERVED_EXTENSION_FLAGS = 0b1111_0100;
const TYPE_KIND = 1 << 0;
const CLOCK_KIND = 1 << 1;
const RESERVED_KINDS = 0b1111_1100;
// The next override ordinal that ends the chain.
const NO_OVERRIDE = 0xffffffff;

const TIMESTAMP_SIZE = DataType.TIMESTAMP.density.valueOf();
const ALIGNMENT_SIZE = 8;
const MULTIPLE_SIZE = 4;
const DATA_LENGTH_SIZE = 4;
const KEY_SIZE = 4;
const SEQ_NUM_SIZE = 4;
const FLAGS_SIZE = 1;

/** The values that the frames of one state agree on, as one side sees them. */
interface Agreement {
  typesDeclared: boolean;
  // dropped is true once a frame of the state failed to decode.
  dropped: boolean;
  dataTypes: DataType[];
  // The agreed clock of each requested key. Zero is no clock.
  clocks: number[];
  declared: channel.Key[];
  // The ascending union of the requested and the declared keys.
  all: channel.Key[];
}

interface CodecState {
  keys: channel.Key[];
  index: Map<channel.Key, number>;
  enc: Agreement;
  dec: Agreement;
}

const needsTypeOverride = (agreed: DataType, dt: DataType): boolean =>
  !agreed.equals(dt) && (!agreed.isInt64Like || !dt.isInt64Like);

const isOwnClock = (key: channel.Key, s: SeriesPayload): boolean =>
  (s.clock ?? 0) === key && s.dataType.isInt64Like;

// Returns the ascending union of a and b, which are ascending and disjoint.
const union = (a: channel.Key[], b: channel.Key[]): channel.Key[] => {
  if (b.length === 0) return a;
  const out: channel.Key[] = [];
  let [i, j] = [0, 0];
  while (i < a.length && j < b.length) out.push(a[i] < b[j] ? a[i++] : b[j++]);
  return out.concat(a.slice(i), b.slice(j));
};

const keysEqual = (a: channel.Key[], b: channel.Key[]): boolean =>
  a.length === b.length && a.every((k, i) => k === b[i]);

const codeOf = (key: channel.Key, dt: DataType): number => {
  const { code } = dt;
  if (code === DataTypeCode.unknown)
    throw new ValidationError(
      `data type "${dt.toString()}" of channel ${key} has no type code`,
    );
  return code;
};

const newAgreement = (keys: channel.Key[], dataTypes: DataType[]): Agreement => ({
  typesDeclared: false,
  dropped: false,
  dataTypes,
  clocks: new Array<number>(keys.length).fill(0),
  declared: [],
  all: keys,
});

interface Entry {
  key: channel.Key;
  series: SeriesPayload;
  // The position of key in the requested keys, or -1 for a declared clock.
  index: number;
}

/** The extension flags, type codes, overrides, and declared clocks of one frame. */
interface Plan {
  ext: number;
  codes: number[];
  kinds: number[];
  clocks: number[];
  declared: channel.Key[];
  all: channel.Key[];
}

/** Reads a frame, failing on any read past its end. */
class FrameReader {
  index: number;
  readonly view: DataView;
  readonly src: Uint8Array;

  constructor(src: Uint8Array, offset: number) {
    this.src = src;
    this.view = new DataView(src.buffer, src.byteOffset, src.byteLength);
    this.index = offset;
  }

  get remaining(): number {
    return this.view.byteLength - this.index;
  }

  private need(n: number): void {
    if (n > this.remaining)
      throw new ValidationError("remote sent a frame that ends early");
  }

  uint8(): number {
    this.need(1);
    return this.view.getUint8(this.index++);
  }

  uint32(): number {
    this.need(4);
    const v = this.view.getUint32(this.index, true);
    this.index += 4;
    return v;
  }

  uint64(): bigint {
    this.need(8);
    const v = this.view.getBigUint64(this.index, true);
    this.index += 8;
    return v;
  }

  bytes(n: number): Uint8Array<ArrayBuffer> {
    this.need(n);
    const b = this.src.slice(this.index, this.index + n);
    this.index += n;
    return b;
  }

  dataType(): DataType {
    const code = this.uint8();
    const dt = DataType.fromCode(code);
    if (dt.equals(DataType.UNKNOWN))
      throw new ValidationError(
        `remote sent data type code ${code}, which this codec does not know`,
      );
    return dt;
  }
}

const formatBits = (b: number): string => b.toString(2).padStart(8, "0");

export class Codec {
  contentType: string = CONTENT_TYPE;
  private states: Map<number, CodecState> = new Map();
  private currState: CodecState | undefined;
  private seqNum: number = 0;
  // oldest is the lowest sequence number that states can hold.
  private oldest: number = 0;

  constructor(keys: channel.Key[] = [], dataTypes: DataType[] = []) {
    if (keys.length > 0 || dataTypes.length > 0) this.update(keys, dataTypes);
  }

  /**
   * Starts a new state of the given keys and their data types. A key named twice counts
   * once. The data types are the encoder's agreed types; the decoder reads its own from
   * the first frame of the state.
   */
  update(keys: channel.Key[], dataTypes: DataType[]): void {
    this.seqNum++;
    // Encoder and decoder walk this order positionally when a frame carries every
    // channel, so it has to match the Core's.
    const sorted = [...new Set(keys)].sort(compareKeys);
    const index = new Map(sorted.map((k, i) => [k, i]));
    const agreed = new Array<DataType>(sorted.length);
    keys.forEach((k, i) => (agreed[index.get(k) as number] = dataTypes[i]));
    const state: CodecState = {
      keys: sorted,
      index,
      enc: newAgreement(sorted, agreed),
      dec: newAgreement(sorted, new Array<DataType>(sorted.length)),
    };
    this.states.set(this.seqNum, state);
    this.currState = state;
  }

  private throwIfNotUpdated(op: string): void {
    if (this.seqNum < 1)
      throw new ValidationError(`
      The codec has not been updated with a list of channels and data types.
      Please call the update method before calling ${op}.
      `);
  }

  // Returns the series of src in key order, keeping the producer's order within a key.
  private entries(src: Payload, state: CodecState): Entry[] {
    const entries = src.keys.map((key, i): Entry => {
      const series = src.series[i];
      const index = state.index.get(key);
      if (index != null) return { key, series, index };
      if (isOwnClock(key, series)) return { key, series, index: -1 };
      throw new ValidationError(
        `Channel ${key} was not provided in the list of channels when opening the writer`,
      );
    });
    return entries.sort((a, b) => compareKeys(a.key, b.key));
  }

  // Plans the extension of the next frame of ag without changing ag.
  private plan(entries: Entry[], state: CodecState): Plan {
    const ag = state.enc;
    const plan: Plan = {
      ext: 0,
      codes: [],
      kinds: [],
      clocks: ag.clocks,
      declared: ag.declared,
      all: ag.all,
    };
    if (!ag.typesDeclared) {
      plan.ext |= TYPES_DECLARED;
      plan.codes = ag.dataTypes.map((dt, i) => codeOf(state.keys[i], dt));
    }
    const overrides = entries.some(
      ({ series, index }) =>
        index >= 0 &&
        ((series.clock ?? 0) !== ag.clocks[index] ||
          needsTypeOverride(ag.dataTypes[index], series.dataType)),
    );
    if (overrides) {
      plan.ext |= OVERRIDES_PRESENT;
      plan.clocks = [...ag.clocks];
      plan.kinds = entries.map(({ key, series, index }) => {
        if (index < 0) return 0;
        let kinds = 0;
        if (needsTypeOverride(ag.dataTypes[index], series.dataType)) {
          codeOf(key, series.dataType);
          kinds |= TYPE_KIND;
        }
        const clock = series.clock ?? 0;
        if (clock !== plan.clocks[index]) {
          kinds |= CLOCK_KIND;
          plan.clocks[index] = clock;
        }
        return kinds;
      });
    }
    if (entries.some(({ index }) => index < 0) || ag.declared.length > 0) {
      const named = (k: channel.Key): boolean => plan.clocks.includes(k);
      const inFrame = new Set(entries.filter((e) => e.index < 0).map((e) => e.key));
      const declared = [
        ...inFrame,
        ...ag.declared.filter((k) => !inFrame.has(k) && named(k)),
      ];
      declared.sort(compareKeys);
      if (!keysEqual(declared, ag.declared)) {
        plan.ext |= CLOCKS_DECLARED;
        plan.declared = declared;
        plan.all = union(state.keys, declared);
      }
    }
    return plan;
  }

  encode(payload: unknown, startOffset: number = 0): Uint8Array<ArrayBuffer> {
    this.throwIfNotUpdated("encode");
    let src = payload as Payload;
    if (payload != null && typeof payload === "object" && "toPayload" in payload)
      src = (payload as Frame).toPayload();
    const state = this.currState as CodecState;
    const entries = this.entries(src, state);
    const plan = this.plan(entries, state);
    let currDataSize = -1;
    let startTime = 0n;
    let endTime = 0n;
    let currAlignment = 0n;
    let byteArraySize = startOffset + FLAGS_SIZE + SEQ_NUM_SIZE;
    let equalLengthsFlag = true;
    let equalTimeRangesFlag = true;
    let equalAlignmentsFlag = true;
    let multiplesPresentFlag = false;
    const channelFlag = keysEqual(
      entries.map((e) => e.key),
      plan.all,
    );
    if (!channelFlag) byteArraySize += entries.length * KEY_SIZE;

    entries.forEach(({ series }) => {
      const pldLength = seriesPldLength(series);
      const start = series.timeRange?.start.valueOf() ?? 0n;
      const end = series.timeRange?.end.valueOf() ?? 0n;
      const alignment = BigInt(series.alignment ?? 0n);
      byteArraySize += seriesWireByteLength(series);
      if (series.dataType.isVariable) equalLengthsFlag = false;
      if ((series.alignmentMultiple ?? 1n) > 1n) multiplesPresentFlag = true;
      if (currDataSize === -1) {
        [currDataSize, startTime, endTime, currAlignment] = [
          pldLength,
          start,
          end,
          alignment,
        ];
        return;
      }
      if (currDataSize !== pldLength) equalLengthsFlag = false;
      if (startTime !== start || endTime !== end) equalTimeRangesFlag = false;
      if (currAlignment !== alignment) equalAlignmentsFlag = false;
    });
    const timeRangesZeroFlag =
      equalTimeRangesFlag && startTime === 0n && endTime === 0n;
    const zeroAlignmentsFlag = equalAlignmentsFlag && currAlignment === 0n;

    if (equalLengthsFlag) byteArraySize += DATA_LENGTH_SIZE;
    else byteArraySize += entries.length * DATA_LENGTH_SIZE;
    if (!timeRangesZeroFlag)
      if (equalTimeRangesFlag) byteArraySize += TIMESTAMP_SIZE * 2;
      else byteArraySize += entries.length * TIMESTAMP_SIZE * 2;
    if (!zeroAlignmentsFlag)
      if (equalAlignmentsFlag) byteArraySize += ALIGNMENT_SIZE;
      else byteArraySize += entries.length * ALIGNMENT_SIZE;
    if (multiplesPresentFlag) byteArraySize += entries.length * MULTIPLE_SIZE;
    if (plan.ext !== 0) byteArraySize += 1 + plan.codes.length;
    if ((plan.ext & CLOCKS_DECLARED) !== 0)
      byteArraySize += KEY_SIZE + KEY_SIZE * plan.declared.length;
    if ((plan.ext & OVERRIDES_PRESENT) !== 0) byteArraySize += 4;
    plan.kinds.forEach((kinds) => {
      if (kinds === 0) return;
      byteArraySize += 1 + 4;
      if ((kinds & TYPE_KIND) !== 0) byteArraySize += 1;
      if ((kinds & CLOCK_KIND) !== 0) byteArraySize += 4;
    });

    const buffer = new Uint8Array(byteArraySize);
    const view = new DataView(buffer.buffer);
    let offset = startOffset;
    buffer[offset] =
      (Number(plan.ext !== 0) << EXTENDED_FLAG_POS) |
      (Number(multiplesPresentFlag) << MULTIPLES_PRESENT_FLAG_POS) |
      (Number(zeroAlignmentsFlag) << ZERO_ALIGNMENTS_FLAG_POS) |
      (Number(equalAlignmentsFlag) << EQUAL_ALIGNMENTS_FLAG_POS) |
      (Number(equalLengthsFlag) << EQUAL_LENGTHS_FLAG_POS) |
      (Number(equalTimeRangesFlag) << EQUAL_TIME_RANGES_FLAG_POS) |
      (Number(timeRangesZeroFlag) << TIME_RANGES_ZERO_FLAG_POS) |
      (Number(channelFlag) << ALL_CHANNELS_PRESENT_FLAG_POS);
    offset++;
    view.setUint32(offset, this.seqNum, true);
    offset += SEQ_NUM_SIZE;
    const u32 = (v: number): void => {
      view.setUint32(offset, v, true);
      offset += 4;
    };
    if (plan.ext !== 0) {
      buffer[offset++] = plan.ext;
      plan.codes.forEach((code) => (buffer[offset++] = code));
    }
    if ((plan.ext & CLOCKS_DECLARED) !== 0) {
      u32(plan.declared.length);
      plan.declared.forEach(u32);
    }
    if (equalLengthsFlag) u32(currDataSize);
    if (equalTimeRangesFlag && !timeRangesZeroFlag) {
      view.setBigUint64(offset, startTime, true);
      view.setBigUint64(offset + TIMESTAMP_SIZE, endTime, true);
      offset += TIMESTAMP_SIZE * 2;
    }
    if (equalAlignmentsFlag && !zeroAlignmentsFlag) {
      view.setBigUint64(offset, currAlignment, true);
      offset += ALIGNMENT_SIZE;
    }
    const nextOverride = (from: number): number => {
      for (let i = from; i < plan.kinds.length; i++) if (plan.kinds[i] !== 0) return i;
      return NO_OVERRIDE;
    };
    if ((plan.ext & OVERRIDES_PRESENT) !== 0) u32(nextOverride(0));

    entries.forEach(({ key, series }, i) => {
      if (!channelFlag) u32(key);
      const kinds = plan.kinds[i] ?? 0;
      if (kinds !== 0) {
        buffer[offset++] = kinds;
        if ((kinds & TYPE_KIND) !== 0) buffer[offset++] = series.dataType.code;
        if ((kinds & CLOCK_KIND) !== 0) u32(series.clock ?? 0);
        u32(nextOverride(i + 1));
      }
      if (!equalLengthsFlag)
        u32(
          series.dataType.isVariable ? series.data.byteLength : seriesPldLength(series),
        );
      const bytes = series.dataType.equals(DataType.BOOLEAN)
        ? packBoolBits(series.data)
        : new Uint8Array(series.data);
      buffer.set(bytes, offset);
      offset += bytes.byteLength;
      if (!equalTimeRangesFlag) {
        view.setBigUint64(offset, series.timeRange?.start.valueOf() ?? 0n, true);
        view.setBigUint64(
          offset + TIMESTAMP_SIZE,
          series.timeRange?.end.valueOf() ?? 0n,
          true,
        );
        offset += TIMESTAMP_SIZE * 2;
      }
      if (!equalAlignmentsFlag) {
        view.setBigUint64(offset, series.alignment ?? 0n, true);
        offset += ALIGNMENT_SIZE;
      }
      if (multiplesPresentFlag) u32(Number(series.alignmentMultiple ?? 1n));
    });
    const ag = state.enc;
    ag.typesDeclared = true;
    [ag.clocks, ag.declared, ag.all] = [plan.clocks, plan.declared, plan.all];
    return buffer;
  }

  /**
   * Decodes a frame. A frame that fails to decode drops its state, so every later frame
   * of the state fails.
   */
  decode(data: Uint8Array | ArrayBuffer, offset: number = 0): Payload {
    this.throwIfNotUpdated("decode");
    const src = data instanceof Uint8Array ? data : new Uint8Array(data);
    const r = new FrameReader(src, offset);
    const flags = r.uint8();
    const seqNum = r.uint32();
    const state = this.states.get(seqNum);
    if (state == null)
      throw new ValidationError(`remote sent invalid sequence number ${seqNum}`);
    // The remote never encodes with an older state again.
    for (; this.oldest < seqNum; this.oldest++) this.states.delete(this.oldest);
    if (state.dec.dropped)
      throw new ValidationError(
        `remote sent a frame for state ${seqNum} after a frame of it failed to decode`,
      );
    let decoded = false;
    try {
      const fr = this.decodeFrame(r, flags, seqNum, state);
      decoded = true;
      return fr;
    } finally {
      if (!decoded) state.dec.dropped = true;
    }
  }

  private decodeFrame(
    r: FrameReader,
    flags: number,
    seqNum: number,
    state: CodecState,
  ): Payload {
    const bit = (pos: number): boolean => ((flags >> pos) & 1) === 1;
    const multiplesPresentFlag = bit(MULTIPLES_PRESENT_FLAG_POS);
    const zeroAlignmentsFlag = bit(ZERO_ALIGNMENTS_FLAG_POS);
    const equalAlignmentsFlag = bit(EQUAL_ALIGNMENTS_FLAG_POS);
    const sizeFlag = bit(EQUAL_LENGTHS_FLAG_POS);
    const equalTimeRangesFlag = bit(EQUAL_TIME_RANGES_FLAG_POS);
    const timeRangesZeroFlag = bit(TIME_RANGES_ZERO_FLAG_POS);
    const channelFlag = bit(ALL_CHANNELS_PRESENT_FLAG_POS);
    const ag = state.dec;
    let ext = 0;
    if (bit(EXTENDED_FLAG_POS)) {
      ext = r.uint8();
      if (ext === 0) throw new ValidationError("remote sent extension flags of 0");
      if ((ext & RESERVED_EXTENSION_FLAGS) !== 0)
        throw new ValidationError(
          `remote sent extension flags ${formatBits(ext)}, which hold a reserved bit`,
        );
    }
    if ((ext & TYPES_DECLARED) === 0) {
      if (!ag.typesDeclared)
        throw new ValidationError(
          `remote sent no type declaration for state ${seqNum}, so it runs an older ` +
            "codec, which this codec does not support",
        );
    } else {
      if (ag.typesDeclared)
        throw new ValidationError(`remote declared the types of state ${seqNum} twice`);
      for (let i = 0; i < state.keys.length; i++) ag.dataTypes[i] = r.dataType();
      ag.typesDeclared = true;
    }
    if ((ext & CLOCKS_DECLARED) !== 0) {
      const n = r.uint32();
      if (n > Math.floor(r.remaining / KEY_SIZE))
        throw new ValidationError("remote sent a frame that ends early");
      const declared: channel.Key[] = [];
      for (let i = 0; i < n; i++) {
        const key = r.uint32();
        if (state.index.has(key))
          throw new ValidationError(
            `remote declared clock ${key}, which the request names`,
          );
        const prev = declared[i - 1];
        if (i > 0 && key <= prev)
          throw new ValidationError(
            `remote declared clock ${key} after clock ${prev}, so the list is not ascending`,
          );
        declared.push(key);
      }
      ag.declared = declared;
      ag.all = union(state.keys, declared);
    }
    const sharedLength = sizeFlag ? r.uint32() : 0;
    let timeRange = new TimeRange({ start: 0n, end: 0n });
    if (equalTimeRangesFlag && !timeRangesZeroFlag)
      timeRange = new TimeRange({ start: r.uint64(), end: r.uint64() });
    let alignment = 0n;
    if (equalAlignmentsFlag && !zeroAlignmentsFlag) alignment = r.uint64();
    let next = (ext & OVERRIDES_PRESENT) !== 0 ? r.uint32() : NO_OVERRIDE;
    let ordinal = 0;

    const readKinds = (): number => {
      const kinds = r.uint8();
      if (kinds === 0)
        throw new ValidationError(
          `remote sent override kinds of 0 for series ${ordinal}`,
        );
      if ((kinds & RESERVED_KINDS) !== 0)
        throw new ValidationError(
          `remote sent override kinds ${formatBits(kinds)} for series ${ordinal}, ` +
            "which hold a reserved bit",
        );
      return kinds;
    };

    const agreementOf = (key: channel.Key): [DataType, number] => {
      const i = state.index.get(key);
      if (i == null) {
        if (!ag.declared.includes(key))
          throw new ValidationError(`unknown channel key: ${key}`);
        if (ordinal === next) {
          readKinds();
          throw new ValidationError(
            `remote sent an override for declared clock ${key}`,
          );
        }
        return [DataType.TIMESTAMP, key];
      }
      let dataType = ag.dataTypes[i];
      if (ordinal === next) {
        const kinds = readKinds();
        if ((kinds & TYPE_KIND) !== 0) dataType = r.dataType();
        if ((kinds & CLOCK_KIND) !== 0) ag.clocks[i] = r.uint32();
        next = r.uint32();
        if (next !== NO_OVERRIDE && next <= ordinal)
          throw new ValidationError(
            `remote sent next override ${next}, which is not after series ${ordinal}`,
          );
      }
      return [dataType, ag.clocks[i]];
    };

    const returnFrame: Payload = { keys: [], series: [] };
    const decodeSeries = (key: channel.Key): void => {
      const [dataType, clock] = agreementOf(key);
      ordinal++;
      if (sizeFlag && dataType.isVariable)
        throw new ValidationError(
          `remote sent variable data type ${dataType.toString()} under a shared length`,
        );
      const size = sizeFlag ? sharedLength : r.uint32();
      const isBool = dataType.equals(DataType.BOOLEAN);
      let byteLength = size;
      if (isBool) byteLength = bitPackedByteCount(size);
      else if (!dataType.isVariable) byteLength *= dataType.density.valueOf();
      const wire = r.bytes(byteLength);
      const series: SeriesPayload = {
        dataType,
        data: isBool ? unpackBoolBits(wire, size) : wire.buffer,
        timeRange,
        alignment,
        clock,
      };
      if (!equalTimeRangesFlag)
        series.timeRange = new TimeRange({ start: r.uint64(), end: r.uint64() });
      if (!equalAlignmentsFlag) series.alignment = r.uint64();
      if (multiplesPresentFlag) {
        const multiple = r.uint32();
        if (multiple > 1) series.alignmentMultiple = BigInt(multiple);
      }
      returnFrame.series.push(series);
      returnFrame.keys.push(key);
    };

    if (channelFlag) {
      ag.all.forEach(decodeSeries);
      if (r.remaining > 0)
        throw new ValidationError("remote sent bytes after the last series");
    } else while (r.remaining > 0) decodeSeries(r.uint32());
    if (next !== NO_OVERRIDE)
      throw new ValidationError(
        `remote ended the frame with the override for series ${next} pending`,
      );
    return returnFrame;
  }
}

const LOW_PERF_SPECIAL_CHAR = 254;
const LOW_PERF_SPECIAL_CHAR_BUF = new Uint8Array([LOW_PERF_SPECIAL_CHAR]);
const HIGH_PERF_SPECIAL_CHAR = 255;
const HIGH_PERF_SPECIAL_CHAR_BUF = new Uint8Array([HIGH_PERF_SPECIAL_CHAR]);
const CONTENT_TYPE = "application/vnd.synnax.frame";

export class WSWriterCodec implements binary.Codec {
  contentType = CONTENT_TYPE;
  private base: Codec;
  private lowPerfCodec: binary.Codec;

  constructor(base: Codec) {
    this.base = base;
    this.lowPerfCodec = binary.JSON_CODEC;
  }

  encode(payload: unknown): Uint8Array<ArrayBuffer> {
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

  decode<P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    const dv = new DataView(data instanceof Uint8Array ? data.buffer : data);
    const codec = dv.getUint8(0);
    if (codec === LOW_PERF_SPECIAL_CHAR)
      return this.lowPerfCodec.decode(data.slice(1), schema, opts);
    const v: WebsocketMessage<WriteRequest> = { type: "data" };
    const frame = this.base.decode(data, 1);
    v.payload = { command: WriterCommand.Write, frame };
    return v as z.infer<P>;
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

  encode(payload: unknown): Uint8Array<ArrayBuffer> {
    return this.lowPerfCodec.encode(payload);
  }

  decode<P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    const dv = new DataView(data instanceof Uint8Array ? data.buffer : data);
    const codec = dv.getUint8(0);
    if (codec === LOW_PERF_SPECIAL_CHAR)
      return this.lowPerfCodec.decode(data.slice(1), schema, opts);
    const v: WebsocketMessage<StreamerResponse> = {
      type: "data",
      payload: { frame: this.base.decode(data, 1) },
    };
    return v as z.infer<P>;
  }
}

export class WSIteratorCodec implements binary.Codec {
  contentType = CONTENT_TYPE;
  private base: Codec;
  private lowPerfCodec: binary.Codec;

  constructor(base: Codec) {
    this.base = base;
    this.lowPerfCodec = binary.JSON_CODEC;
  }

  encode(payload: unknown): Uint8Array<ArrayBuffer> {
    return this.lowPerfCodec.encode(payload);
  }

  decode<P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    const dv = new DataView(data instanceof Uint8Array ? data.buffer : data);
    const codec = dv.getUint8(0);
    if (codec === LOW_PERF_SPECIAL_CHAR)
      return this.lowPerfCodec.decode(data.slice(1), schema, opts);
    const v: WebsocketMessage<unknown> = {
      type: "data",
      payload: {
        variant: IteratorResponseVariant.Data,
        ack: false,
        command: 0,
        frame: this.base.decode(data, 1),
      },
    };
    return v as z.infer<P>;
  }
}
