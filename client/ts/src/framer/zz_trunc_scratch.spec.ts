// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync, writeFileSync } from "fs";

import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Codec } from "@/framer/codec";

describe("scratch trunc", () => {
  it("should sweep", () => {
    const { cases } = JSON.parse(
      readFileSync(
        new URL("../../../../x/testdata/framecodec/vectors.json", import.meta.url),
        "utf-8",
      ),
    ) as {
      cases: Array<{
        name: string;
        keys: number[];
        data_types: string[];
        frames: Array<{ encoded: string }>;
      }>;
    };
    const fromHex = (h: string): Uint8Array<ArrayBuffer> =>
      new Uint8Array(h.match(/../g)?.map((b) => parseInt(b, 16)) ?? []);
    const counts = new Map<string, number>();
    for (const c of cases)
      c.frames.forEach((f, fi) => {
        const full = fromHex(f.encoded);
        for (let cut = 0; cut < full.length; cut++) {
          const dec = new Codec(
            c.keys,
            c.data_types.map((d) => new DataType(d)),
          );
          c.frames.slice(0, fi).forEach((p) => dec.decode(fromHex(p.encoded)));
          let k: string;
          try {
            const d = dec.decode(full.slice(0, cut));
            k = `OK ${c.name} ${fi} ${cut} ${JSON.stringify(d.keys)}`;
          } catch (e) {
            k = (e as Error).message;
          }
          counts.set(k, (counts.get(k) ?? 0) + 1);
        }
      });
    writeFileSync(process.env.TRUNC_TS ?? "/dev/null", JSON.stringify([...counts.entries()]));
    expect(counts.size).toBeGreaterThan(0);
  });
});
