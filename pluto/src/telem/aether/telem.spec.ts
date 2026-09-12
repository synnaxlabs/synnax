// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { zod } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { telem } from "@/telem/aether";

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

  it("pipeline", async () => {
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
    expect(await p.value()).toBe(true);
  });

  describe("AbstractSource loading", () => {
    class LatchedSource extends telem.AbstractSource<z.ZodNumber> {
      schema = z.number();

      value(): number {
        return this.props;
      }

      latch(): void {
        this.loading_ = true;
      }

      settle(): void {
        this.declareLoaded();
      }
    }

    it("should not report loading by default", () => {
      expect(new LatchedSource(1).loading()).toBe(false);
    });

    it("should notify exactly once when a latched source settles", () => {
      const source = new LatchedSource(1);
      const handleChange = vi.fn();
      source.onChange(handleChange);
      source.latch();
      expect(source.loading()).toBe(true);
      source.settle();
      source.settle();
      expect(source.loading()).toBe(false);
      expect(handleChange).toHaveBeenCalledTimes(1);
    });

    it("should not notify when settling an unlatched source", () => {
      const source = new LatchedSource(1);
      const handleChange = vi.fn();
      source.onChange(handleChange);
      source.settle();
      expect(handleChange).not.toHaveBeenCalled();
    });
  });
});
