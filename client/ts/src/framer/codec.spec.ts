import { WebsocketMessage } from "@synnaxlabs/freighter";
import { DataType, Series } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

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
    it("should e + d an identical frame to the codec", () => {
      const codec = new Codec(
        [1, 2, 3],
        [DataType.FLOAT32, DataType.FLOAT32, DataType.FLOAT32],
      );
      const fr = new framer.Frame(
        [1, 2, 3],
        [
          new Series(new Float32Array([1, 2, 3])),
          new Series(new Float32Array([4, 5, 6])),
          new Series(new Float32Array([7, 8, 9])),
        ],
      );
      const encoded = codec.encode(fr.toPayload());
      const decoded = new framer.Frame(codec.decode(encoded));
      expect(decoded.series[0].data).toEqual(new Float32Array([1, 2, 3]));
      expect(decoded.series[1].data).toEqual(new Float32Array([4, 5, 6]));
      expect(decoded.series[2].data).toEqual(new Float32Array([7, 8, 9]));
    });
    it("should e + d a frame with different lengths for its series", () => {
      const codec = new Codec(
        [1, 2, 3],
        [DataType.FLOAT32, DataType.FLOAT32, DataType.FLOAT32],
      );
      const fr = new framer.Frame(
        [1, 2, 3],
        [
          new Series(new Float32Array([1, 2, 3])),
          new Series(new Float32Array([4, 5, 6])),
          new Series(new Float32Array([7, 8])),
        ],
      );
      const encoded = codec.encode(fr.toPayload());
      const decoded = new framer.Frame(codec.decode(encoded));
      expect(decoded.series[0].data).toEqual(new Float32Array([1, 2, 3]));
      expect(decoded.series[1].data).toEqual(new Float32Array([4, 5, 6]));
      expect(decoded.series[2].data).toEqual(new Float32Array([7, 8]));
    });
    it("should e + d a frame with different keys", () => {
      const codec = new Codec(
        [1, 2, 3],
        [DataType.FLOAT32, DataType.FLOAT32, DataType.FLOAT32],
      );
      const fr = new framer.Frame(
        [1, 2],
        [
          new Series(new Float32Array([1, 2, 3])),
          new Series(new Float32Array([4, 5, 6])),
        ],
      );
      const encoded = codec.encode(fr.toPayload());
      const decoded = new framer.Frame(codec.decode(encoded));
      expect(decoded.series[0].data).toEqual(new Float32Array([1, 2, 3]));
      expect(decoded.series[1].data).toEqual(new Float32Array([4, 5, 6]));
    });
    it("should e + d a frame with skipped keys", () => {
      const codec = new Codec([1, 3], [DataType.FLOAT32, DataType.FLOAT32]);
      const fr = new framer.Frame(
        [1, 3],
        [
          new Series(new Float32Array([1, 2, 3])),
          new Series(new Float32Array([4, 5, 6])),
        ],
      );
      const encoded = codec.encode(fr.toPayload());
      const decoded = new framer.Frame(codec.decode(encoded));
      expect(decoded.series[0].data).toEqual(new Float32Array([1, 2, 3]));
      expect(decoded.series[1].data).toEqual(new Float32Array([4, 5, 6]));
      expect(decoded.keys).toEqual([1, 3]);
    });
    // test.only("performance", () => {
    //   const fr = new framer.Frame(
    //     [1, 2, 3],
    //     [
    //       new Series(new Float32Array([1, 2, 3])),
    //       new Series(new Float32Array([4, 5, 6])),
    //       new Series(new Float32Array([7, 8, 9])),
    //     ],
    //   );
    //   const pld = fr.toPayload();
    //   const CODECS: binary.Codec[] = [
    //     binary.JSON_CODEC,
    //     new Codec([1, 2, 3], [DataType.FLOAT32, DataType.FLOAT32, DataType.FLOAT32]),
    //   ];
    //   const ITERS = 100_000;
    //   CODECS.forEach((codec) => {
    //     const start = performance.now();
    //     for (let i = 0; i < ITERS; i++) {
    //       codec.encode(pld);
    //     }
    //     const end = performance.now();
    //     console.log(codec.constructor.name, end - start);
    //   });

    // });
  });
  describe.only("writer codec", () => {
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
