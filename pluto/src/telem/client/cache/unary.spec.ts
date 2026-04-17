// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import {
  MultiSeries,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NATIVE_FIDELITY, Unary } from "@/telem/client/cache/unary";

const buildChannel = (): channel.Payload =>
  new channel.Channel({
    key: 1,
    name: "ch",
    dataType: DataType.FLOAT32,
    isIndex: false,
  });

const buildSeries = (args: {
  timeRange: TimeRange;
  alignment: bigint;
  alignmentMultiple?: bigint;
  values: number[];
}) =>
  new Series({
    data: new Float32Array(args.values),
    dataType: DataType.FLOAT32,
    timeRange: args.timeRange,
    alignment: args.alignment,
    alignmentMultiple: args.alignmentMultiple,
  });

describe("Unary fidelity-aware cache", () => {
  describe("native-fidelity backward compatibility", () => {
    it("read() returns series written via writeStatic()", () => {
      const u = new Unary({ channel: buildChannel(), dynamicBufferSize: 100 });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      u.writeStatic(
        new MultiSeries([buildSeries({ timeRange: tr, alignment: 0n, values: [1] })]),
      );
      const { series, gaps } = u.read(tr);
      expect(series.series).toHaveLength(1);
      expect(gaps).toHaveLength(0);
    });
  });

  describe("layered cross-tier lookup", () => {
    it("coarse request is satisfied by a finer cached tier", () => {
      const u = new Unary({ channel: buildChannel(), dynamicBufferSize: 100 });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      // Write native-fidelity data (finer than any fidelity > 1n).
      u.writeStaticAtFidelity(
        new MultiSeries([buildSeries({ timeRange: tr, alignment: 0n, values: [1] })]),
        NATIVE_FIDELITY,
      );
      // Request at a coarser fidelity; the native tier qualifies because
      // alignmentMultiple 1n <= 8n.
      const { series, gaps } = u.dirtyReadAtFidelity(tr, 8n);
      expect(series.series).toHaveLength(1);
      expect(gaps).toHaveLength(0);
    });

    it("fine request is NOT satisfied by coarser cached data", () => {
      const u = new Unary({ channel: buildChannel(), dynamicBufferSize: 100 });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      // Write a coarse series (fidelity 8n).
      u.writeStaticAtFidelity(
        new MultiSeries([
          buildSeries({ timeRange: tr, alignment: 0n, alignmentMultiple: 8n, values: [1] }),
        ]),
        8n,
      );
      // Request at native fidelity; the coarse tier does NOT qualify.
      const { series, gaps } = u.dirtyReadAtFidelity(tr, NATIVE_FIDELITY);
      expect(series.series).toHaveLength(0);
      expect(gaps).toHaveLength(1);
    });

    it("prefers the coarsest qualifying tier when multiple tiers cover the range", () => {
      const u = new Unary({ channel: buildChannel(), dynamicBufferSize: 100 });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      // Fine tier covers the range.
      u.writeStaticAtFidelity(
        new MultiSeries([buildSeries({ timeRange: tr, alignment: 0n, values: [1, 2, 3] })]),
        NATIVE_FIDELITY,
      );
      // Coarse tier also covers the range.
      u.writeStaticAtFidelity(
        new MultiSeries([
          buildSeries({ timeRange: tr, alignment: 0n, alignmentMultiple: 4n, values: [1] }),
        ]),
        4n,
      );
      // Request at fidelity 8n; both tiers qualify. Coarsest-first walk picks
      // the 4n tier, which returns a single-sample series; the native tier is
      // never consulted because the remaining gaps are empty.
      const { series, gaps } = u.dirtyReadAtFidelity(tr, 8n);
      expect(series.series).toHaveLength(1);
      expect(series.series[0].length).toEqual(1);
      expect(gaps).toHaveLength(0);
    });

    it("falls through coarser to finer to fill gaps", () => {
      const u = new Unary({ channel: buildChannel(), dynamicBufferSize: 100 });
      // Coarse tier covers [1s, 2s).
      u.writeStaticAtFidelity(
        new MultiSeries([
          buildSeries({
            timeRange: TimeStamp.seconds(1).spanRange(TimeSpan.seconds(1)),
            alignment: 0n,
            alignmentMultiple: 4n,
            values: [1],
          }),
        ]),
        4n,
      );
      // Fine tier covers [2s, 3s).
      u.writeStaticAtFidelity(
        new MultiSeries([
          buildSeries({
            timeRange: TimeStamp.seconds(2).spanRange(TimeSpan.seconds(1)),
            alignment: 0n,
            values: [2],
          }),
        ]),
        NATIVE_FIDELITY,
      );
      // Requesting [1s, 3s) at fidelity 4n should use both tiers, no gaps.
      const { series, gaps } = u.dirtyReadAtFidelity(
        TimeStamp.seconds(1).spanRange(TimeSpan.seconds(2)),
        4n,
      );
      expect(series.series).toHaveLength(2);
      expect(gaps).toHaveLength(0);
    });
  });

  describe("fidelity normalization", () => {
    it("treats fidelity 0n as native fidelity on write and read", () => {
      const u = new Unary({ channel: buildChannel(), dynamicBufferSize: 100 });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      u.writeStaticAtFidelity(
        new MultiSeries([buildSeries({ timeRange: tr, alignment: 0n, values: [1] })]),
        0n,
      );
      // Reading with fidelity 0n should also normalize and find the data in
      // the native tier.
      const { series, gaps } = u.dirtyReadAtFidelity(tr, 0n);
      expect(series.series).toHaveLength(1);
      expect(gaps).toHaveLength(0);
    });
  });
});
