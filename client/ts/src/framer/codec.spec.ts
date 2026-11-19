// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { type WebsocketMessage } from "@synnaxlabs/freighter";
import { DataType, Series, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { type channel } from "@/channel";
import { framer } from "@/framer";
import { Codec, WSIteratorCodec, WSStreamerCodec, WSWriterCodec } from "@/framer/codec";
import { ValidationError } from "@/index";

// Wire bytes the Core uses to mark JSON and binary WebSocket messages.
const LOW_PERF_SPECIAL_CHAR = 254;
const HIGH_PERF_SPECIAL_CHAR = 255;

describe("encoder", () => {
  describe("base codec", () => {
    interface Spec {
      name: string;
      channels: channel.Key[];
      dataTypes: DataType[];
      frame: framer.Frame;
    }

    const SPECS: Spec[] = [
      {
        name: "All Channels Present, In Order",
        channels: [1, 2, 3],
        dataTypes: [DataType.INT64, DataType.FLOAT32, DataType.FLOAT64],
        frame: new framer.Frame(
          [1, 2, 3],
          [
            new Series(new BigInt64Array([1n, 2n, 3n])),
            new Series(new Float32Array([4, 5, 6])),
            new Series(new Float64Array([7, 8, 9])),
          ],
        ),
      },
      {
        name: "All Channels Present, Out of Order",
        channels: [3, 1, 2],
        dataTypes: [DataType.FLOAT64, DataType.INT64, DataType.FLOAT32],
        frame: new framer.Frame(
          [2, 3, 1],
          [
            new Series(new Float32Array([4, 5, 6])),
            new Series(new Float64Array([7, 8, 9])),
            new Series(new BigInt64Array([1n, 2n, 3n])),
          ],
        ),
      },
      {
        name: "Some Channels Present, In Order",
        channels: [1, 2, 3],
        dataTypes: [DataType.UINT8, DataType.FLOAT32, DataType.FLOAT64],
        frame: new framer.Frame(
          [1, 3],
          [
            new Series(new Uint8Array([1, 2, 3])),
            new Series(new Float64Array([7, 8, 9])),
          ],
        ),
      },
      {
        name: "Some Channels Present, Out of Order",
        channels: [1, 2, 3],
        dataTypes: [DataType.UINT8, DataType.FLOAT32, DataType.FLOAT64],
        frame: new framer.Frame(
          [3, 1],
          [
            new Series(new Float64Array([7, 8, 9])),
            new Series(new Uint8Array([1, 2, 3])),
          ],
        ),
      },
      {
        name: "Only One Channel Present",
        channels: [1, 2, 3, 4, 5],
        dataTypes: [
          DataType.UINT8,
          DataType.UINT8,
          DataType.UINT8,
          DataType.UINT8,
          DataType.UINT8,
        ],
        frame: new framer.Frame([3], [new Series(new Uint8Array([1, 2, 3, 4, 5]))]),
      },
      {
        name: "All Same Time Range",
        channels: [1, 2],
        dataTypes: [DataType.UINT8, DataType.FLOAT32],
        frame: new framer.Frame(
          [1, 2],
          [
            new Series({
              dataType: DataType.UINT8,
              data: new Uint8Array([1]),
              timeRange: new TimeStamp(0).spanRange(5),
            }),
            new Series({
              dataType: DataType.FLOAT32,
              data: new Float32Array([1, 2, 3, 4]),
              timeRange: new TimeStamp(0).spanRange(5),
            }),
          ],
        ),
      },
      {
        name: "All Same Non-Zero Time Range",
        channels: [1, 2],
        dataTypes: [DataType.UINT8, DataType.FLOAT32],
        frame: new framer.Frame(
          [1, 2],
          [
            new Series({
              dataType: DataType.UINT8,
              data: new Uint8Array([1]),
              timeRange: new TimeStamp(1000000000).spanRange(5000000000),
            }),
            new Series({
              dataType: DataType.FLOAT32,
              data: new Float32Array([1, 2, 3, 4]),
              timeRange: new TimeStamp(1000000000).spanRange(5000000000),
            }),
          ],
        ),
      },
      {
        name: "Different Non-Zero Time Ranges",
        channels: [1, 2],
        dataTypes: [DataType.UINT8, DataType.FLOAT32],
        frame: new framer.Frame(
          [1, 2],
          [
            new Series({
              dataType: DataType.UINT8,
              data: new Uint8Array([1]),
              timeRange: new TimeStamp(1000000000).spanRange(5000000000),
            }),
            new Series({
              dataType: DataType.FLOAT32,
              data: new Float32Array([1, 2, 3, 4]),
              timeRange: new TimeStamp(2000000000).spanRange(3000000000),
            }),
          ],
        ),
      },
      {
        name: "Partial Present, Different Lengths",
        channels: [1, 2, 3],
        dataTypes: [DataType.UINT8, DataType.FLOAT32, DataType.FLOAT64],
        frame: new framer.Frame(
          [1, 3],
          [new Series(new Uint8Array([1])), new Series(new Float64Array([1, 2, 3, 4]))],
        ),
      },
      {
        name: "Same Alignments",
        channels: [1, 2],
        dataTypes: [DataType.UINT8, DataType.FLOAT32],
        frame: new framer.Frame(
          [1, 2],
          [
            new Series({
              dataType: DataType.UINT8,
              data: new Uint8Array([1]),
              alignment: 5n,
            }),
            new Series({
              dataType: DataType.FLOAT32,
              data: new Uint8Array([1, 2, 3, 4]),
              alignment: 5n,
            }),
          ],
        ),
      },
      {
        name: "Different Alignments",
        channels: [1, 2],
        dataTypes: [DataType.UINT8, DataType.FLOAT32],
        frame: new framer.Frame(
          [1, 2],
          [
            new Series({
              dataType: DataType.UINT8,
              data: new Uint8Array([1]),
              alignment: 5n,
            }),
            new Series({
              dataType: DataType.FLOAT32,
              data: new Uint8Array([1, 2, 3, 4]),
              alignment: 10n,
            }),
          ],
        ),
      },
      // Keys are (12-bit node key << 20 | 20-bit local key), so their decimal widths
      // differ across nodes. These two pairs sort differently under a lexicographic
      // comparator than under a numeric one, which is the order the Core encodes in.
      {
        name: "All Channels Present, Keys Spanning A Decimal Width",
        // Node 9 (7 digits) and node 10 (8 digits).
        channels: [9437185, 10485761],
        dataTypes: [DataType.INT64, DataType.FLOAT32],
        frame: new framer.Frame(
          [9437185, 10485761],
          [
            new Series(new BigInt64Array([1n, 2n, 3n])),
            new Series(new Float32Array([4, 5, 6])),
          ],
        ),
      },
      {
        name: "All Channels Present, Free Channel Alongside A Leased One",
        // Node 5, and a free channel (node key 4095).
        channels: [5242881, 4293918721],
        dataTypes: [DataType.INT64, DataType.FLOAT32],
        frame: new framer.Frame(
          [5242881, 4293918721],
          [
            new Series(new BigInt64Array([1n, 2n, 3n])),
            new Series(new Float32Array([4, 5, 6])),
          ],
        ),
      },
      {
        name: "Variable Data Types",
        channels: [1, 2, 3],
        dataTypes: [DataType.UINT8, DataType.STRING, DataType.JSON],
        frame: new framer.Frame(
          [1, 2, 3],
          [
            new Series(new Uint8Array([1])),
            new Series({
              data: ["one", "two", "three"],
              dataType: DataType.STRING,
            }),
            new Series({
              data: [{ a: 1 }, { b: 2 }, { c: 3 }],
              dataType: DataType.JSON,
            }),
          ],
        ),
      },
      {
        name: "Bool Single Sample",
        channels: [1],
        dataTypes: [DataType.BOOLEAN],
        frame: new framer.Frame(
          [1],
          [new Series({ data: [true], dataType: DataType.BOOLEAN })],
        ),
      },
      {
        name: "Bool Exact Byte Boundary",
        channels: [1],
        dataTypes: [DataType.BOOLEAN],
        frame: new framer.Frame(
          [1],
          [
            new Series({
              data: [true, false, true, false, true, false, true, false],
              dataType: DataType.BOOLEAN,
            }),
          ],
        ),
      },
      {
        name: "Bool One Past Byte Boundary",
        channels: [1],
        dataTypes: [DataType.BOOLEAN],
        frame: new framer.Frame(
          [1],
          [
            new Series({
              data: [true, false, true, false, true, false, true, false, true],
              dataType: DataType.BOOLEAN,
            }),
          ],
        ),
      },
      {
        name: "Bool Seven Samples Partial Last Byte",
        channels: [1],
        dataTypes: [DataType.BOOLEAN],
        frame: new framer.Frame(
          [1],
          [
            new Series({
              data: [true, false, true, true, false, false, true],
              dataType: DataType.BOOLEAN,
            }),
          ],
        ),
      },
      {
        name: "Bool Mixed With Other Types",
        channels: [1, 2, 3],
        dataTypes: [DataType.BOOLEAN, DataType.FLOAT32, DataType.UINT8],
        frame: new framer.Frame(
          [1, 2, 3],
          [
            new Series({
              data: [true, false, true],
              dataType: DataType.BOOLEAN,
            }),
            new Series(new Float32Array([1.5, 2.5, 3.5])),
            new Series(new Uint8Array([7, 8, 9])),
          ],
        ),
      },
    ];

    SPECS.forEach((spec) => {
      it(`should encode & decode ${spec.name}`, () => {
        const codec = new Codec(spec.channels, spec.dataTypes);
        const encoded = codec.encode(spec.frame.toPayload());
        const decoded = codec.decode(encoded);
        decoded.keys.forEach((k, i) => {
          const dcs = decoded.series[i];
          const ser = spec.frame.get(k);
          expect(ser.series.length).toBeGreaterThan(0);
          const os = ser.series[0];
          if (dcs.timeRange != null && !dcs.timeRange.span.isZero)
            expect(dcs.timeRange.toString()).toEqual(os.timeRange?.toString());
          expect(new Series(dcs).toString()).toEqual(os.toString());
        });
      });
    });
  });

  describe("series views", () => {
    it("should encode only the samples of a sub-series", () => {
      const codec = new Codec([1], [DataType.INT32]);
      const sub = new Series(new Int32Array([1, 2, 3, 4, 5])).sub(1, 3);
      const decoded = codec.decode(codec.encode(new framer.Frame([1], [sub])));
      expect(Array.from(new Series(decoded.series[0]))).toEqual([2, 3]);
    });

    it("should encode only the written samples of an allocated series", () => {
      const codec = new Codec([1], [DataType.FLOAT64]);
      const s = Series.alloc({ capacity: 10, dataType: DataType.FLOAT64 });
      s.write(new Series(new Float64Array([1.5, 2.5])));
      const decoded = codec.decode(codec.encode(new framer.Frame([1], [s])));
      expect(Array.from(new Series(decoded.series[0]))).toEqual([1.5, 2.5]);
    });

    it("should encode only the samples of a subarray", () => {
      const codec = new Codec([1], [DataType.UINT8]);
      const data = new Uint8Array([9, 8, 7, 6]).subarray(2);
      const decoded = codec.decode(
        codec.encode(new framer.Frame([1], [new Series(data)])),
      );
      expect(Array.from(new Series(decoded.series[0]))).toEqual([7, 6]);
    });
  });

  describe("alignment multiples", () => {
    const MULTIPLES_PRESENT_BIT = 1 << 6;
    const EXTENDED_BIT = 1 << 7;

    it("should round-trip the alignment multiple of each series", () => {
      const codec = new Codec([1, 2], [DataType.FLOAT32, DataType.FLOAT32]);
      const encoded = codec.encode(
        new framer.Frame(
          [1, 2],
          [
            new Series({
              data: new Float32Array([1, 2]),
              alignment: 8n,
              alignmentMultiple: 4n,
            }),
            new Series({ data: new Float32Array([3, 4]), alignment: 12n }),
          ],
        ),
      );
      expect(encoded[0] & MULTIPLES_PRESENT_BIT).toEqual(MULTIPLES_PRESENT_BIT);
      const decoded = new framer.Frame(codec.decode(encoded));
      expect(decoded.series[0].alignment).toEqual(8n);
      expect(decoded.series[0].alignmentMultiple).toEqual(4n);
      expect(decoded.series[1].alignment).toEqual(12n);
      expect(decoded.series[1].alignmentMultiple).toEqual(1n);
      expect(Array.from(decoded.series[1])).toEqual([3, 4]);
    });

    it("should leave the multiples flag unset when every multiple is one", () => {
      const codec = new Codec([1], [DataType.FLOAT32]);
      const encoded = codec.encode(
        new framer.Frame([1], [new Series(new Float32Array([1, 2]))]),
      );
      expect(encoded[0] & MULTIPLES_PRESENT_BIT).toEqual(0);
    });

    it("should set the extended bit in the first frame of a state", () => {
      const codec = new Codec([1], [DataType.FLOAT32]);
      const encoded = codec.encode(
        new framer.Frame([1], [new Series(new Float32Array([1, 2]))]),
      );
      expect(encoded[0] & EXTENDED_BIT).toEqual(EXTENDED_BIT);
    });
  });

  describe("dynamic codec", () => {
    it("should allow the caller to update the codec", () => {
      const codec = new Codec();
      codec.update([1], [DataType.INT32]);
      const encoded = codec.encode(
        new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]),
      );
      const decoded1 = new framer.Frame(codec.decode(encoded));
      expect(Array.from(decoded1.series[0])).toEqual([1, 2, 3]);
      expect(decoded1.keys[0]).toEqual(1);
      codec.update([2], [DataType.INT64]);
      const encoded2 = codec.encode(
        new framer.Frame([2], [new Series(new BigInt64Array([1n, 2n, 3n]))]),
      );
      const decoded2 = new framer.Frame(codec.decode(encoded2));
      expect(Array.from(decoded2.series[0])).toEqual([1n, 2n, 3n]);
      expect(decoded2.keys[0]).toEqual(2);
    });

    it("should throw an error if the codec is not initialized", () => {
      const codec = new Codec();
      expect(() =>
        codec.encode(new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))])),
      ).toThrow(ValidationError);
    });

    it("should use the correct encode/decode state even if the codecs are out of sync", () => {
      const encoder = new Codec();
      const decoder = new Codec();
      encoder.update([1], [DataType.INT32]);
      decoder.update([1], [DataType.INT32]);

      const fr = new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]);
      let encoded = encoder.encode(fr);
      let decoded = new framer.Frame(decoder.decode(encoded));
      expect(decoded.keys[0]).toEqual(1);
      expect(decoded.series[0].data).toEqual(fr.series[0].data);

      decoder.update([2], [DataType.INT64]);
      encoded = encoder.encode(fr);
      decoded = new framer.Frame(decoder.decode(encoded));
      expect(decoded.keys[0]).toEqual(1);
      expect(decoded.series[0].data).toEqual(fr.series[0].data);

      encoder.update([2], [DataType.INT64]);
      expect(() => encoder.encode(fr)).toThrow(ValidationError);
      const fr2 = new framer.Frame([2], [new Series(new BigInt64Array([1n, 2n, 3n]))]);
      encoded = encoder.encode(fr2);
      decoded = new framer.Frame(decoder.decode(encoded));
      expect(decoded.keys[0]).toEqual(2);
      expect(decoded.series[0].data).toEqual(fr2.series[0].data);
    });

    it("should reject a frame encoded with keys older than the last frame decoded", () => {
      const encoder = new Codec();
      const decoder = new Codec();
      encoder.update([1], [DataType.INT32]);
      decoder.update([1], [DataType.INT32]);
      const fr = new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]);
      const stale = encoder.encode(fr);
      encoder.update([1, 2], [DataType.INT32, DataType.INT64]);
      decoder.update([1, 2], [DataType.INT32, DataType.INT64]);
      decoder.decode(encoder.encode(fr));
      expect(() => decoder.decode(stale)).toThrow(
        new ValidationError("remote sent invalid sequence number 1"),
      );
    });
  });

  describe("websocket codec JSON fallback", () => {
    [WSWriterCodec, WSStreamerCodec, WSIteratorCodec].forEach((WSCodec) => {
      it(`should label parse errors in ${WSCodec.name}`, () => {
        const codec = new WSCodec(new Codec([1], [DataType.INT32]));
        const json = new TextEncoder().encode(JSON.stringify({ id: "a" }));
        const data = new Uint8Array([LOW_PERF_SPECIAL_CHAR, ...json]);
        const schema = z.object({ id: z.number() });
        expect(() => codec.decode(data, schema, { label: "test response" })).toThrow(
          /test response/,
        );
      });
    });
  });

  describe("websocket writer codec", () => {
    it("should correctly encode and decode a websocket write request", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const fr = new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]);
      const writeReq: framer.WriteRequest = {
        command: framer.WriterCommand.Write,
        frame: fr.toPayload(),
      };
      const msg: WebsocketMessage<framer.WriteRequest> = {
        type: "data",
        payload: writeReq,
      };
      const codec = new WSWriterCodec(baseCodec);
      const encoded = codec.encode(msg);
      expect(encoded[0]).toEqual(HIGH_PERF_SPECIAL_CHAR);
      const decoded = codec.decode(encoded) as WebsocketMessage<framer.WriteRequest>;
      expect(decoded.payload?.command).toEqual(framer.WriterCommand.Write);
      const decodedFr = new framer.Frame(decoded.payload?.frame);
      expect(decodedFr.series[0].data).toEqual(fr.series[0].data);
    });

    it("should correctly e +d a WS write set authority request", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const writeReq: framer.WriteRequest = {
        command: framer.WriterCommand.SetAuthority,
        config: { authorities: [123] },
      };
      const codec = new WSWriterCodec(baseCodec);
      const msg: WebsocketMessage<framer.WriteRequest> = {
        type: "data",
        payload: writeReq,
      };
      const encoded = codec.encode(msg);
      expect(encoded[0]).toEqual(LOW_PERF_SPECIAL_CHAR);
      const decoded = codec.decode(encoded);
      expect(decoded).toEqual(msg);
    });
  });

  describe("websocket iterator codec", () => {
    it("should JSON-encode an iterator request without a special char prefix", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const codec = new WSIteratorCodec(baseCodec);
      const msg: WebsocketMessage<unknown> = {
        type: "data",
        payload: { command: 1, span: 0 },
      };
      const encoded = codec.encode(msg);
      // Iterator requests are sent as raw JSON; no special-char prefix.
      // Confirm the first byte is JSON ('{' or '[').
      expect([0x7b, 0x5b]).toContain(encoded[0]);
    });

    it("should binary-decode a data variant response and synthesize variant=Data", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const fr = new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]);
      const codec = new WSIteratorCodec(baseCodec);
      const encoded = new Uint8Array(baseCodec.encode(fr.toPayload(), 1));
      encoded[0] = HIGH_PERF_SPECIAL_CHAR;
      const decoded = codec.decode(
        encoded,
      ) as WebsocketMessage<framer.IteratorResponse>;
      expect(decoded.payload?.variant).toEqual(framer.IteratorResponseVariant.Data);
      const decodedFr = new framer.Frame(decoded.payload?.frame);
      expect(decodedFr.series[0].data).toEqual(fr.series[0].data);
    });

    it("should JSON-decode an ack variant response prefixed with the low-perf char", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const codec = new WSIteratorCodec(baseCodec);
      const ackMsg: WebsocketMessage<framer.IteratorResponse> = {
        type: "data",
        payload: {
          variant: framer.IteratorResponseVariant.Ack,
          ack: true,
          command: framer.IteratorCommand.Next,
        },
      };
      // Simulate what the server sends: LOW_PERF_SPECIAL_CHAR + JSON-encoded message.
      const json = new TextEncoder().encode(JSON.stringify(ackMsg));
      const encoded = new Uint8Array(json.byteLength + 1);
      encoded[0] = LOW_PERF_SPECIAL_CHAR;
      encoded.set(json, 1);
      const decoded = codec.decode(
        encoded,
      ) as WebsocketMessage<framer.IteratorResponse>;
      expect(decoded.payload?.variant).toEqual(framer.IteratorResponseVariant.Ack);
      expect(decoded.payload?.ack).toEqual(true);
    });
  });

  describe("golden frames", () => {
    // The vectors are shared with the Go, Python, and C++ codecs, so that all four
    // encode and decode each frame byte for byte.
    const VECTORS_URL = new URL(
      "../../../../x/testdata/framecodec/vectors.json",
      import.meta.url,
    );
    interface VectorSeries {
      key: number;
      data_type: string;
      data: string;
      clock?: number;
    }
    interface VectorCase {
      name: string;
      keys: number[];
      data_types: string[];
      frames: Array<{ name: string; series: VectorSeries[]; encoded: string }>;
    }
    const { cases } = JSON.parse(readFileSync(VECTORS_URL, "utf-8")) as {
      cases: VectorCase[];
    };
    const fromHex = (h: string): Uint8Array<ArrayBuffer> =>
      new Uint8Array(h.match(/../g)?.map((b) => parseInt(b, 16)) ?? []);
    const toHex = (b: Uint8Array): string =>
      Array.from(b, (v) => v.toString(16).padStart(2, "0")).join("");
    const frameOf = (series: VectorSeries[]): framer.Frame =>
      new framer.Frame(
        series.map((s) => s.key),
        series.map(
          (s) =>
            new Series({
              data: fromHex(s.data).buffer,
              dataType: s.data_type,
              clock: s.clock ?? 0,
            }),
        ),
      );

    it("should hold every case", () => expect(cases.length).toBeGreaterThan(0));

    cases.forEach((c) =>
      it(`should encode and decode ${c.name} byte for byte`, () => {
        const dataTypes = c.data_types.map((dt) => new DataType(dt));
        const enc = new Codec(c.keys, dataTypes);
        const dec = new Codec(c.keys, dataTypes);
        c.frames.forEach((f) => {
          const encoded = enc.encode(frameOf(f.series).toPayload());
          expect(toHex(encoded), f.name).toEqual(f.encoded);
          const decoded = new framer.Frame(dec.decode(fromHex(f.encoded)));
          expect(decoded.keys, f.name).toEqual(f.series.map((s) => s.key));
          decoded.series.forEach((s, i) => {
            const want = f.series[i];
            expect(s.dataType.toString(), f.name).toEqual(want.data_type);
            expect(s.clock, f.name).toEqual(want.clock ?? 0);
            expect(toHex(new Uint8Array(s.buffer)), f.name).toEqual(want.data);
          });
        });
      }),
    );
  });

  describe("agreement", () => {
    const hex = (h: string): Uint8Array<ArrayBuffer> =>
      new Uint8Array(
        h
          .replaceAll(" ", "")
          .match(/../g)
          ?.map((b) => parseInt(b, 16)) ?? [],
      );
    const float32Codec = (): Codec => new Codec([1], [DataType.FLOAT32]);
    // The first frame of a float32 key 1 whose one sample, 1.0, has clock 5. Byte 5 is
    // the extension flags, byte 6 the type code, byte 15 the override kinds, and bytes
    // 20 to 23 the next override ordinal.
    const CLOCK_FRAME =
      "BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F";

    it("should label an int64 series on a timestamp key with no override", () => {
      const enc = new Codec([1], [DataType.TIMESTAMP]);
      const encoded = enc.encode(
        new framer.Frame([1], [new Series(new BigInt64Array([7n]))]),
      );
      expect(encoded).toEqual(hex("BF 01000000 01 0B 01000000 0700000000000000"));
      const dec = new Codec([1], [DataType.TIMESTAMP]);
      expect(dec.decode(encoded).series[0].dataType.equals(DataType.TIMESTAMP)).toBe(
        true,
      );
    });

    it("should carry a type override for its own series only", () => {
      const enc = float32Codec();
      const dec = float32Codec();
      const roundTrip = (s: Series): Series =>
        new framer.Frame(dec.decode(enc.encode(new framer.Frame([1], [s])))).series[0];
      expect(roundTrip(new Series(new Float64Array([1]))).dataType).toEqual(
        DataType.FLOAT64,
      );
      expect(roundTrip(new Series(new Float32Array([2]))).dataType).toEqual(
        DataType.FLOAT32,
      );
    });

    it("should commit nothing when a frame fails to encode", () => {
      const c = float32Codec();
      // A Series of an unknown type cannot be built, so the payload is built by hand.
      const bad = {
        keys: [1],
        series: [
          {
            data: new Uint8Array(4).buffer,
            dataType: new DataType("int128"),
            clock: 5,
          },
        ],
      };
      expect(() => c.encode(bad)).toThrow(
        new ValidationError('data type "int128" of channel 1 has no type code'),
      );
      const good = new Series({ data: new Float32Array([1]), clock: 5 });
      expect(c.encode(new framer.Frame([1], [good]))).toEqual(hex(CLOCK_FRAME));
    });

    const REJECTS: Array<
      [string, (b: Uint8Array<ArrayBuffer>) => Uint8Array<ArrayBuffer>, string]
    > = [
      [
        "a reserved extension bit",
        (b) => ((b[5] |= 0b0000_0100), b),
        "remote sent extension flags 00000111, which hold a reserved bit",
      ],
      [
        "a second extension byte",
        (b) => ((b[5] |= 0b1000_0000), b),
        "remote sent extension flags 10000011, which hold a reserved bit",
      ],
      [
        "extension flags of 0",
        (b) => ((b[5] = 0), b),
        "remote sent extension flags of 0",
      ],
      [
        "type code 0",
        (b) => ((b[6] = 0), b),
        "remote sent data type code 0, which this codec does not know",
      ],
      [
        "an unknown type code",
        (b) => ((b[6] = 17), b),
        "remote sent data type code 17, which this codec does not know",
      ],
      [
        "a variable type under a shared length",
        (b) => ((b[6] = 13), b),
        "remote sent variable data type string under a shared length",
      ],
      [
        "override kinds of 0",
        (b) => ((b[15] = 0), b),
        "remote sent override kinds of 0 for series 0",
      ],
      [
        "a reserved override kind",
        (b) => ((b[15] |= 0b0000_0100), b),
        "remote sent override kinds 00000110 for series 0, which hold a reserved bit",
      ],
      [
        "a next override that is not after the series",
        (b) => (new DataView(b.buffer).setUint32(20, 0, true), b),
        "remote sent next override 0, which is not after series 0",
      ],
      [
        "an override pending at the end of the frame",
        (b) => (new DataView(b.buffer).setUint32(20, 1, true), b),
        "remote ended the frame with the override for series 1 pending",
      ],
      [
        "bytes after the last series",
        (b) => new Uint8Array([...b, 0]),
        "remote sent bytes after the last series",
      ],
    ];
    REJECTS.forEach(([name, corrupt, msg]) =>
      it(`should reject ${name} and drop the state`, () => {
        const dec = float32Codec();
        expect(() => dec.decode(corrupt(hex(CLOCK_FRAME)))).toThrow(
          new ValidationError(msg),
        );
        expect(() => dec.decode(hex(CLOCK_FRAME))).toThrow(
          new ValidationError(
            "remote sent a frame for state 1 after a frame of it failed to decode",
          ),
        );
      }),
    );

    it("should reject a second type declaration for a state", () => {
      const dec = float32Codec();
      dec.decode(hex(CLOCK_FRAME));
      expect(() => dec.decode(hex(CLOCK_FRAME))).toThrow(
        new ValidationError("remote declared the types of state 1 twice"),
      );
    });

    it("should name the older codec of a remote that declares no types", () => {
      expect(() => float32Codec().decode(hex("3F 01000000 01000000 0000803F"))).toThrow(
        new ValidationError(
          "remote sent no type declaration for state 1, so it runs an older codec, " +
            "which this codec does not support",
        ),
      );
    });

    describe("declared clocks", () => {
      const WORKED_EXAMPLE =
        "BF 01000000 0B 0901 01000000 05000000 02000000 01000000" +
        " E803000000000000 D007000000000000" +
        " 03 0A 05000000 FFFFFFFF 000000000000F03F 0000000000000040 0100";
      const workedCodec = (): Codec =>
        new Codec([6, 8], [DataType.FLOAT32, DataType.UINT8]);

      it("should reject a requested key in the list", () => {
        const b = hex(WORKED_EXAMPLE);
        new DataView(b.buffer).setUint32(12, 6, true);
        expect(() => workedCodec().decode(b)).toThrow(
          new ValidationError("remote declared clock 6, which the request names"),
        );
      });

      it("should reject an override on a declared clock", () => {
        const b = hex(WORKED_EXAMPLE);
        new DataView(b.buffer).setUint32(20, 0, true);
        const inserted = new Uint8Array([
          ...b.slice(0, 24),
          0b0000_0010,
          ...b.slice(24),
        ]);
        expect(() => workedCodec().decode(inserted)).toThrow(
          new ValidationError("remote sent an override for declared clock 5"),
        );
      });

      it("should reject a list that is not ascending", () => {
        const c = new Codec([6], [DataType.FLOAT32]);
        const encoded = c.encode(
          new framer.Frame(
            [5, 6, 7],
            [
              new Series({
                data: new BigInt64Array([1n]),
                dataType: DataType.TIMESTAMP,
                clock: 5,
              }),
              new Series(new Float32Array([1])),
              new Series({
                data: new BigInt64Array([1n]),
                dataType: DataType.TIMESTAMP,
                clock: 7,
              }),
            ],
          ),
        );
        const view = new DataView(encoded.buffer);
        expect(view.getUint32(7, true)).toEqual(2);
        view.setUint32(11, 7, true);
        view.setUint32(15, 5, true);
        expect(() => c.decode(encoded)).toThrow(
          new ValidationError(
            "remote declared clock 5 after clock 7, so the list is not ascending",
          ),
        );
      });

      it("should fail on a list that the frame does not hold", () => {
        const b = hex(WORKED_EXAMPLE);
        new DataView(b.buffer).setUint32(8, 0xffffffff, true);
        expect(() => workedCodec().decode(b)).toThrow(
          new ValidationError("remote sent a frame that ends early"),
        );
      });

      it("should throw on a key that is not requested and not its own clock", () => {
        const c = new Codec([6], [DataType.FLOAT32]);
        expect(() =>
          c.encode(new framer.Frame([9], [new Series(new Float32Array([1]))])),
        ).toThrow(ValidationError);
      });
    });
  });
});
