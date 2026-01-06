// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { observe } from "@/observe";

describe("observe", () => {
  describe("Observer", () => {
    it("should correctly notify the receives of values", () => {
      const observer = new observe.Observer<number>();
      const values: number[] = [];
      observer.onChange((value) => values.push(value));
      observer.notify(1);
      observer.notify(2);
      expect(values).toEqual([1, 2]);
    });
    it("should stop notifying a handler after the destructor is called", () => {
      const observer = new observe.Observer<number>();
      const values: number[] = [];
      const destructor = observer.onChange((value) => values.push(value));
      observer.notify(1);
      destructor();
      observer.notify(2);
      expect(values).toEqual([1]);
    });
    it("should correctly transform an observed value", () => {
      const observer = new observe.Observer<number, string>((value) => [
        value.toString(),
        true,
      ]);
      const values: string[] = [];
      observer.onChange((value) => values.push(value));
      observer.notify(1);
      observer.notify(2);
      expect(values).toEqual(["1", "2"]);
    });
    it("should not notify when the transform returns false", () => {
      const observer = new observe.Observer<number, string>((value) => [
        value.toString(),
        false,
      ]);
      const values: string[] = [];
      observer.onChange((value) => values.push(value));
      observer.notify(1);
      observer.notify(2);
      expect(values).toEqual([]);
    });
  });
});
