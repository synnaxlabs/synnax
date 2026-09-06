// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Context, MemoizedSource } from "@/telem/aether/context";
import { createFactory } from "@/telem/aether/factory";
import { fixedNumber } from "@/telem/aether/static";
import { type Source } from "@/telem/aether/telem";

describe("MemoizedSource", () => {
  const provider = (): Context => new Context(createFactory());

  const stub = (loading?: () => boolean): Source<number> => ({
    value: () => 1,
    onChange: () => () => {},
    loading,
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
