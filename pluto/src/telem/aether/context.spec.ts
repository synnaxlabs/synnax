// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Context, MemoizedSource } from "@/telem/aether/context";
import { createFactory } from "@/telem/aether/factory";
import { fixedNumber } from "@/telem/aether/static";
import { type Source, type ValueProps } from "@/telem/aether/telem";

describe("MemoizedSource", () => {
  const provider = (): Context => new Context(createFactory());

  const stub = (
    loading?: () => boolean,
    sampleTime?: () => TimeStamp | null,
    fetching?: () => boolean,
  ): Source<number> => ({
    value: () => 1,
    onChange: () => () => {},
    loading,
    sampleTime,
    fetching,
  });

  describe("value", () => {
    it("should forward the value props to the wrapped source", () => {
      const received: (ValueProps | undefined)[] = [];
      const wrapped: Source<number> = {
        value: (props) => {
          received.push(props);
          return 1;
        },
        onChange: () => () => {},
      };
      const source = new MemoizedSource(wrapped, provider(), fixedNumber(1));
      const props: ValueProps = {
        view: { bounds: { lower: 1, upper: 2 }, width: 100 },
      };
      source.value(props);
      source.value();
      expect(received).toEqual([props, undefined]);
    });
  });

  describe("fetching", () => {
    it("should forward fetching from the wrapped source", () => {
      const source = new MemoizedSource(
        stub(undefined, undefined, () => true),
        provider(),
        fixedNumber(1),
      );
      expect(source.fetching()).toBe(true);
    });

    it("should default to false when the wrapped source lacks fetching", () => {
      const source = new MemoizedSource(stub(), provider(), fixedNumber(1));
      expect(source.fetching()).toBe(false);
    });
  });

  describe("sampleTime", () => {
    it("should forward sampleTime from the wrapped source", () => {
      const at = TimeStamp.seconds(5);
      const source = new MemoizedSource(
        stub(undefined, () => at),
        provider(),
        fixedNumber(1),
      );
      expect(source.sampleTime()).toBe(at);
    });

    it("should default to null when the wrapped source lacks sampleTime", () => {
      const source = new MemoizedSource(stub(), provider(), fixedNumber(1));
      expect(source.sampleTime()).toBeNull();
    });
  });

  describe("loading", () => {
    it("should forward loading from the wrapped source", () => {
      const source = new MemoizedSource(
        stub(() => true),
        provider(),
        fixedNumber(1),
      );
      expect(source.loading()).toBe(true);
    });

    it("should default to false when the wrapped source lacks loading", () => {
      const source = new MemoizedSource(stub(), provider(), fixedNumber(1));
      expect(source.loading()).toBe(false);
    });
  });
});
