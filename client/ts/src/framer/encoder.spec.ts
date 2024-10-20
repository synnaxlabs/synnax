import { binary, DataType, Series } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { framer } from "@/framer";
import { Codec } from "@/framer/encoder";

describe("encoder", () => {
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
  test.only("performance", () => {
    const fr = new framer.Frame(
      [1, 2, 3],
      [
        new Series(new Float32Array([1, 2, 3])),
        new Series(new Float32Array([4, 5, 6])),
        new Series(new Float32Array([7, 8, 9])),
      ],
    );
    const pld = fr.toPayload();
    const CODECS: binary.Codec[] = [
      binary.JSON_CODEC,
      new Codec([1, 2, 3], [DataType.FLOAT32, DataType.FLOAT32, DataType.FLOAT32]),
    ];
    const ITERS = 100_000;
    CODECS.forEach((codec) => {
      const start = performance.now();
      for (let i = 0; i < ITERS; i++) {
        codec.encode(pld);
      }
      const end = performance.now();
      console.log(codec.constructor.name, end - start);
    });
  });
});
