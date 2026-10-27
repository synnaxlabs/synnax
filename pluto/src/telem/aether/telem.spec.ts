// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { TimeStamp, zod } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { telem } from "@/telem/aether";

const SAMPLE_TIME = TimeStamp.seconds(5);

const resultSource = <V>(result: telem.Result<V>): telem.Source<V> => ({
  value: () => result,
  onChange: () => () => {},
});

const ERROR_STATUS = status.create({ variant: "error", message: "failed" });

class InvalidPropsSource extends telem.AbstractSource<z.ZodNumber> {
  schema = z.number();
  async value(): Promise<number> {
    return this.props;
  }
}

describe("telem", () => {
  it("throws a ParseError when props fail their schema", () => {
    const source = new InvalidPropsSource("nan");
    expect(() => source.props).toThrow(zod.ParseError);
  });

  it("pipeline", () => {
    const s1 = telem.fixedNumber(20);
    const s2 = telem.fixedNumber(9);
    const avg = telem.mean({});
    const bool = telem.withinBounds({ trueBound: { upper: 15, lower: 5 } });
    const p = new telem.SourcePipeline(
      {
        connections: [
          { from: "s1", to: "avg" },
          { from: "s2", to: "avg" },
          { from: "avg", to: "bool" },
        ],
        outlet: "bool",
        segments: { s1, s2, avg, bool },
      },
      telem.createFactory(),
    );
    expect(p.value().data).toBe(true);
  });

  describe("Result", () => {
    const stringifier = (): telem.StringifyNumber =>
      new telem.StringifyNumber({ precision: 2, notation: "standard" });

    it("should copy status, aggregation, time, and appended through a unary transformer", () => {
      const t = stringifier();
      t.setSources({
        s: resultSource({
          data: 1,
          status: telem.LOADING_STATUS,
          aggregation: "average",
          time: SAMPLE_TIME,
          appended: 7,
        }),
      });
      expect(t.value()).toEqual({
        data: "1.00",
        status: telem.LOADING_STATUS,
        aggregation: "average",
        time: SAMPLE_TIME,
        appended: 7,
      });
    });

    it("should report the worst status and the newest time of many sources", () => {
      const m = new telem.Mean({});
      m.setSources({
        a: resultSource({ data: 2, status: ERROR_STATUS, time: TimeStamp.seconds(3) }),
        b: resultSource({ data: 4, status: telem.LOADING_STATUS, time: SAMPLE_TIME }),
        c: resultSource(telem.success(6)),
      });
      expect(m.value()).toEqual({ data: 4, status: ERROR_STATUS, time: SAMPLE_TIME });
    });

    it("should sum the appended counts of its sources", () => {
      const m = new telem.Mean({});
      m.setSources({
        a: resultSource({ ...telem.success(2), appended: 3 }),
        b: resultSource({ ...telem.success(4), appended: 4 }),
      });
      expect(m.value().appended).toBe(7);
    });

    it("should report no appended count when a source has none", () => {
      const m = new telem.Mean({});
      m.setSources({
        a: resultSource({ ...telem.success(2), appended: 3 }),
        b: resultSource(telem.success(6)),
      });
      expect(m.value().appended).toBeUndefined();
    });

    it("should report success and no time when no source has one", () => {
      const m = new telem.Mean({});
      m.setSources({ a: resultSource(telem.success(2)) });
      expect(m.value()).toEqual({
        data: 2,
        status: telem.SUCCESS_STATUS,
        time: undefined,
      });
    });

    it("should forward a pipeline outlet's result", () => {
      const p = new telem.SourcePipeline<string>(
        {
          connections: [{ from: "s", to: "str" }],
          outlet: "str",
          segments: {
            s: telem.fixedNumber(1),
            str: telem.stringifyNumber({ precision: 2, notation: "standard" }),
          },
        },
        telem.createFactory(),
      );
      (p.sources.str as telem.StringifyNumber).setSources({
        s: resultSource({ data: 1, status: telem.SUCCESS_STATUS, time: SAMPLE_TIME }),
      });
      expect(p.value().time).toBe(SAMPLE_TIME);
    });
  });

  describe("SourcePipeline forwarding", () => {
    const pipeline = (outlet: telem.Source<number>): telem.SourcePipeline<number> => {
      const p = new telem.SourcePipeline<number>(
        { connections: [], outlet: "s", segments: { s: telem.fixedNumber(1) } },
        telem.createFactory(),
      );
      p.sources.s = outlet;
      return p;
    };

    it("should forward the value props to its outlet", () => {
      const received: (telem.ValueProps | undefined)[] = [];
      const p = pipeline({
        value: (props) => {
          received.push(props);
          return telem.success(1);
        },
        onChange: () => () => {},
      });
      const props: telem.ValueProps = {
        bounds: { lower: 1, upper: 2 },
        points: 100,
      };
      p.value(props);
      expect(received).toEqual([props]);
    });
  });
});
