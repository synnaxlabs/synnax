// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type WebsocketMessage } from "@synnaxlabs/freighter";
import { DataType, Series, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { ValidationError } from "@/errors";
import { framer } from "@/framer";
import {
  Codec,
  HIGH_PERF_SPECIAL_CHAR,
  LOW_PER_SPECIAL_CHAR,
  WSWriterCodec,
} from "@/framer/codec";
import { Frame } from "@/framer/frame";
import { WriterCommand, type WriteRequest } from "@/framer/writer";

describe("encoder", () => {
  describe("base codec", () => {
    interface Spec {
      name: string;
      channels: channel.Keys;
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
        name: "Different Time Ranges",
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
        frame: new Frame(
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

  describe("dynamic codec", () => {
    it("should allow the caller to update the codec", () => {
      const codec = new Codec();
      codec.update([1], [DataType.INT32]);
      const encoded = codec.encode(
        new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]),
      );
      const decoded1 = new Frame(codec.decode(encoded));
      expect(Array.from(decoded1.series[0])).toEqual([1, 2, 3]);
      expect(decoded1.keys[0]).toEqual(1);
      codec.update([2], [DataType.INT64]);
      const encoded2 = codec.encode(
        new framer.Frame([2], [new Series(new BigInt64Array([1n, 2n, 3n]))]),
      );
      const decoded2 = new Frame(codec.decode(encoded2));
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
      let decoded = new Frame(decoder.decode(encoded));
      expect(decoded.keys[0]).toEqual(1);
      expect(decoded.series[0].data).toEqual(fr.series[0].data);

      decoder.update([2], [DataType.INT64]);
      encoded = encoder.encode(fr);
      decoded = new Frame(decoder.decode(encoded));
      expect(decoded.keys[0]).toEqual(1);
      expect(decoded.series[0].data).toEqual(fr.series[0].data);

      encoder.update([2], [DataType.INT64]);
      expect(() => encoder.encode(fr)).toThrow(ValidationError);
      const fr2 = new framer.Frame([2], [new Series(new BigInt64Array([1n, 2n, 3n]))]);
      encoded = encoder.encode(fr2);
      decoded = new Frame(decoder.decode(encoded));
      expect(decoded.keys[0]).toEqual(2);
      expect(decoded.series[0].data).toEqual(fr2.series[0].data);
    });
  });

  describe("websocket writer codec", () => {
    it("should correctly encode and decode a websocket write request", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const fr = new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]);
      const writeReq: WriteRequest = {
        command: WriterCommand.Write,
        frame: fr.toPayload(),
      };
      const msg: WebsocketMessage<WriteRequest> = { type: "data", payload: writeReq };
      const codec = new WSWriterCodec(baseCodec);
      const encoded = codec.encode(msg);
      expect(encoded[0]).toEqual(HIGH_PERF_SPECIAL_CHAR);
      const decoded = codec.decode(encoded) as WebsocketMessage<WriteRequest>;
      expect(decoded.payload?.command).toEqual(WriterCommand.Write);
      const decodedFr = new Frame(decoded.payload?.frame);
      expect(decodedFr.series[0].data).toEqual(fr.series[0].data);
    });

    it("should correctly e +d a WS write set authority request", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const writeReq: WriteRequest = {
        command: WriterCommand.SetAuthority,
        config: { authorities: [123] },
      };
      const codec = new WSWriterCodec(baseCodec);
      const msg: WebsocketMessage<WriteRequest> = { type: "data", payload: writeReq };
      const encoded = codec.encode(msg);
      expect(encoded[0]).toEqual(LOW_PER_SPECIAL_CHAR);
      const decoded = codec.decode(encoded);
      expect(decoded).toEqual(msg);
    });
  });
});
