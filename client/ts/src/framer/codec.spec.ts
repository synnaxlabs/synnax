import { WebsocketMessage } from "@synnaxlabs/freighter";
import { DataType, Series, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { framer } from "@/framer";
import {
  Codec,
  HIGH_PERF_SPECIAL_CHAR,
  LOW_PER_SPECIAL_CHAR,
  WSWriterCodec,
} from "@/framer/codec";
import { Frame } from "@/framer/frame";
import { WriterCommand, WriteRequest } from "@/framer/writer";

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
          if (dcs.timeRange != null && !dcs.timeRange.isZero)
            expect(dcs.timeRange.toString()).toEqual(os._timeRange?.toString());
          expect(new Series(dcs).toString()).toEqual(os.toString());
        });
      });
    });
  });
  describe("writer codec", () => {
    it("should correctly e + d a WS write request", () => {
      const baseCodec = new Codec([1], [DataType.INT32]);
      const fr = new framer.Frame([1], [new Series(new Int32Array([1, 2, 3]))]);
      const writeReq: WriteRequest = {
        command: WriterCommand.Write,
        frame: fr.toPayload(),
      };
      const msg: WebsocketMessage<WriteRequest> = { type: "data", payload: writeReq };
      const codec = new WSWriterCodec(baseCodec);
      const encoded = codec.encode(msg);
      const dv = new DataView(encoded);
      expect(dv.getUint8(0)).toEqual(HIGH_PERF_SPECIAL_CHAR);
      const decoded = codec.decode<WebsocketMessage<WriteRequest>>(encoded);
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
      const dv = new DataView(encoded);
      expect(dv.getUint8(0)).toEqual(LOW_PER_SPECIAL_CHAR);
      const decoded = codec.decode(encoded);
      expect(decoded).toEqual(msg);
    });
  });
});
