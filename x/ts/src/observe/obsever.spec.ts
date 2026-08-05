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
import { TimeSpan } from "@/telem";

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

  describe("until", () => {
    it("should resolve immediately when the current value satisfies", async () => {
      const observer = new observe.Observer<number>();
      await expect(
        observe.until(
          observer,
          () => 5,
          (v) => v > 3,
        ),
      ).resolves.toBe(5);
    });

    it("should resolve with the first satisfying value", async () => {
      const observer = new observe.Observer<number>();
      const current = 0;
      const promise = observe.until(
        observer,
        () => current,
        (v) => v > 2,
      );
      observer.notify(1);
      observer.notify(3);
      await expect(promise).resolves.toBe(3);
    });

    it("should ignore values that do not satisfy", async () => {
      const observer = new observe.Observer<number>();
      const promise = observe.until(
        observer,
        () => 0,
        (v) => v > 2,
        TimeSpan.milliseconds(10),
      );
      observer.notify(1);
      observer.notify(2);
      await expect(promise).rejects.toThrow(/timed out/);
    });

    it("should reject when the timeout elapses", async () => {
      const observer = new observe.Observer<number>();
      await expect(
        observe.until(
          observer,
          () => 0,
          (v) => v > 2,
          TimeSpan.milliseconds(5),
        ),
      ).rejects.toThrow(/timed out/);
    });

    it("should unsubscribe once settled", async () => {
      const observer = new observe.Observer<number>();
      await observe
        .until(
          observer,
          () => 0,
          (v) => v > 2,
          TimeSpan.milliseconds(5),
        )
        .catch(() => {});
      const values: number[] = [];
      observer.onChange((value) => values.push(value));
      observer.notify(9);
      expect(values).toEqual([9]);
    });
  });
});
