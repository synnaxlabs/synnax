// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Context, MemoizedSource } from "@/telem/aether/context";
import { createFactory } from "@/telem/aether/factory";
import { fixedNumber } from "@/telem/aether/static";
import {
  LOADING_STATUS,
  type Result,
  type Source,
  success,
  type ValueProps,
} from "@/telem/aether/telem";

describe("MemoizedSource", () => {
  const provider = (): Context => new Context(createFactory());

  describe("value", () => {
    it("should forward the value props to the wrapped source", () => {
      const received: (ValueProps | undefined)[] = [];
      const wrapped: Source<number> = {
        value: (props) => {
          received.push(props);
          return success(1);
        },
        onChange: () => () => {},
      };
      const source = new MemoizedSource(wrapped, provider(), fixedNumber(1));
      const props: ValueProps = {
        bounds: { lower: 1, upper: 2 },
        points: 100,
      };
      source.value(props);
      source.value();
      expect(received).toEqual([props, undefined]);
    });

    it("should return the wrapped source's result", () => {
      const result: Result<number> = {
        data: 1,
        status: LOADING_STATUS,
        aggregation: "average",
        time: TimeStamp.seconds(5),
      };
      const wrapped: Source<number> = { value: () => result, onChange: () => () => {} };
      const source = new MemoizedSource(wrapped, provider(), fixedNumber(1));
      expect(source.value()).toBe(result);
    });
  });

  describe("updateInPlace", () => {
    const createWrapped = (update?: Source<number>["update"]): Source<number> => ({
      value: () => success(1),
      onChange: () => () => {},
      update,
    });

    it("should keep a source that takes the new props in place", () => {
      const update = vi.fn(() => true);
      const prov = provider();
      const source = new MemoizedSource(createWrapped(update), prov, fixedNumber(1));
      const next = fixedNumber(2);
      expect(source.updateInPlace(prov, next)).toBe(true);
      expect(update).toHaveBeenCalledWith(next.props);
      expect(source.shouldUpdate(prov, next)).toBe(false);
    });

    it("should not keep a source that refuses the new props", () => {
      const prov = provider();
      const source = new MemoizedSource(
        createWrapped(() => false),
        prov,
        fixedNumber(1),
      );
      expect(source.updateInPlace(prov, fixedNumber(2))).toBe(false);
      expect(source.shouldUpdate(prov, fixedNumber(2))).toBe(true);
    });

    it("should not keep a source with no update", () => {
      const prov = provider();
      const source = new MemoizedSource(createWrapped(), prov, fixedNumber(1));
      expect(source.updateInPlace(prov, fixedNumber(2))).toBe(false);
    });

    it("should not give the props of another provider or type", () => {
      const update = vi.fn(() => true);
      const prov = provider();
      const source = new MemoizedSource(createWrapped(update), prov, fixedNumber(1));
      expect(source.updateInPlace(provider(), fixedNumber(2))).toBe(false);
      expect(source.updateInPlace(prov, { ...fixedNumber(2), type: "other" })).toBe(
        false,
      );
      expect(update).not.toHaveBeenCalled();
    });
  });
});
