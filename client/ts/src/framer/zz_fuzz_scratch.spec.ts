// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync, readFileSync } from "fs";

import { DataType, Series, TimeRange } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { framer } from "@/framer";
import { Codec } from "@/framer/codec";

const PATH = process.env.FUZZ_OUT ?? "";

interface S {
  key: number;
  data_type: string;
  data: string;
  clock?: number;
  time_range?: [number, number];
  alignment?: number;
}
interface D {
  key: number;
  data_type: string;
  data: string;
  clock: number;
  time_range: [number, number];
  alignment: number;
}
interface C {
  name: string;
  keys: number[];
  data_types: string[];
  frames: Array<{
    name: string;
    series: S[];
    encoded: string;
    error?: string;
    decoded?: D[];
  }>;
}

const fromHex = (h: string): Uint8Array<ArrayBuffer> =>
  new Uint8Array(h.match(/../g)?.map((b) => parseInt(b, 16)) ?? []);
const toHex = (b: Uint8Array): string =>
  Array.from(b, (v) => v.toString(16).padStart(2, "0")).join("");

describe("scratch fuzz", () => {
  it("should match go", () => {
    if (PATH === "" || !existsSync(PATH)) return;
    const { cases } = JSON.parse(readFileSync(PATH, "utf-8")) as { cases: C[] };
    const bad: string[] = [];
    for (const c of cases) {
      const dts = c.data_types.map((dt) => new DataType(dt));
      const enc = new Codec(c.keys, dts);
      const dec = new Codec(c.keys, dts);
      for (const f of c.frames) {
        const fr = new framer.Frame(
          f.series.map((s) => s.key),
          f.series.map(
            (s) =>
              new Series({
                data: fromHex(s.data).buffer,
                dataType: s.data_type,
                clock: s.clock ?? 0,
                timeRange: new TimeRange(
                  BigInt(s.time_range?.[0] ?? 0),
                  BigInt(s.time_range?.[1] ?? 0),
                ),
                alignment: BigInt(s.alignment ?? 0),
              }),
          ),
        );
        if (f.error != null && f.error !== "") {
          try {
            enc.encode(fr.toPayload());
            bad.push(`GO ERR TS OK ${c.name} ${f.name} ${f.error}`);
          } catch {
            // both fail
          }
          break;
        }
        let got: string;
        try {
          got = toHex(enc.encode(fr.toPayload()));
        } catch (e) {
          bad.push(`TS ENC ERR ${c.name} ${f.name} ${String(e)}`);
          break;
        }
        if (got !== f.encoded) {
          bad.push(`ENC MISMATCH ${c.name} ${f.name}\n go ${f.encoded}\n ts ${got}`);
          break;
        }
        let d: framer.Frame;
        try {
          d = new framer.Frame(dec.decode(fromHex(f.encoded)));
        } catch (e) {
          bad.push(`TS DEC ERR ${c.name} ${f.name} ${String(e)}`);
          break;
        }
        const want = f.decoded ?? [];
        if (want.length !== d.series.length) {
          bad.push(`DEC LEN ${c.name} ${f.name}`);
          break;
        }
        let stop = false;
        d.series.forEach((s, i) => {
          if (stop) return;
          const w = want[i];
          const ok =
            d.keys[i] === w.key &&
            s.dataType.toString() === w.data_type &&
            s.clock === w.clock &&
            s.timeRange.start.valueOf() === BigInt(w.time_range[0]) &&
            s.timeRange.end.valueOf() === BigInt(w.time_range[1]) &&
            s.alignment === BigInt(w.alignment) &&
            toHex(new Uint8Array(s.buffer)) === w.data;
          if (!ok) {
            bad.push(
              `DEC MISMATCH ${c.name} ${f.name} ${JSON.stringify(w)} ${s.dataType.toString()} ${s.clock} ${s.alignment} ${toHex(new Uint8Array(s.buffer))}`,
            );
            stop = true;
          }
        });
        if (stop) break;
      }
    }
    expect(bad).toEqual([]);
  });
});
