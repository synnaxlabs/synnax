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

import { telem } from "@/telem/aether";

describe("MemoizedSource", () => {
  const provider = (): telem.Context => new telem.Context(telem.createFactory());

  describe("value", () => {
    it("should forward the value props to the wrapped source", () => {
      const received: (telem.ValueProps | undefined)[] = [];
      const wrapped: telem.Source<number> = {
        value: (props) => {
          received.push(props);
          return telem.success(1);
        },
        onChange: () => () => {},
      };
      const source = new telem.MemoizedSource(
        wrapped,
        provider(),
        telem.fixedNumber(1),
      );
      const props: telem.ValueProps = {
        bounds: { lower: 1, upper: 2 },
        points: 100,
      };
      source.value(props);
      source.value();
      expect(received).toEqual([props, undefined]);
    });

    it("should return the wrapped source's result", () => {
      const result: telem.Result<number> = {
        data: 1,
        status: telem.LOADING_STATUS,
        aggregation: "average",
        time: TimeStamp.seconds(5),
      };
      const wrapped: telem.Source<number> = {
        value: () => result,
        onChange: () => () => {},
      };
      const source = new telem.MemoizedSource(
        wrapped,
        provider(),
        telem.fixedNumber(1),
      );
      expect(source.value()).toBe(result);
    });
  });
});
