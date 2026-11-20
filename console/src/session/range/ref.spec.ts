// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { dynamicWindow, staticWindow, TimeRange, TimeSpan, uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createTestRange } from "@/platform/range/testutil";
import { Range } from "@/session/range";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const PERSISTED: Range.PersistedState = { variant: "persisted", key: uuid.create() };

const DYNAMIC: Range.DynamicState = {
  variant: "dynamic",
  key: "dynamic-1",
  name: "Dynamic 1",
  span: 1000,
};

const keyRef = (key: string): ranger.Ref => ({ variant: "key", key });

const windowRef = (span: number): ranger.Ref => ({
  variant: "window",
  window: dynamicWindow(span),
});

const staticRef = (start: number, end: number): ranger.Ref => ({
  variant: "window",
  window: staticWindow(new TimeRange(start, end)),
});

describe("range ref", () => {
  describe("toRef", () => {
    it("should reference a persisted range by its key", () => {
      expect(Range.toRef(PERSISTED)).toEqual(keyRef(PERSISTED.key));
    });

    it("should turn a dynamic range into a rolling window", () => {
      expect(Range.toRef(DYNAMIC)).toEqual(windowRef(1000));
    });
  });

  describe("refsEqual", () => {
    it("should match key references by key", () => {
      expect(Range.refsEqual(keyRef("a"), keyRef("a"))).toBe(true);
      expect(Range.refsEqual(keyRef("a"), keyRef("b"))).toBe(false);
    });

    it("should match rolling windows by span", () => {
      expect(Range.refsEqual(windowRef(1000), windowRef(1000))).toBe(true);
      expect(Range.refsEqual(windowRef(1000), windowRef(2000))).toBe(false);
    });

    it("should match fixed windows by bounds", () => {
      expect(Range.refsEqual(staticRef(0, 1000), staticRef(0, 1000))).toBe(true);
      expect(Range.refsEqual(staticRef(0, 1000), staticRef(0, 2000))).toBe(false);
    });

    it("should not match references of different variants", () => {
      expect(Range.refsEqual(keyRef("a"), windowRef(1000))).toBe(false);
      expect(Range.refsEqual(windowRef(1000), keyRef("a"))).toBe(false);
      expect(Range.refsEqual(windowRef(1000), staticRef(0, 1000))).toBe(false);
      expect(Range.refsEqual(staticRef(0, 1000), windowRef(1000))).toBe(false);
    });
  });

  describe("findKey", () => {
    const states = [PERSISTED, DYNAMIC];

    it("should find the session range a reference shows", () => {
      expect(Range.findKey(windowRef(1000), states)).toBe(DYNAMIC.key);
      expect(Range.findKey(keyRef(PERSISTED.key), states)).toBe(PERSISTED.key);
    });

    it("should return undefined when no session range shows the reference", () => {
      expect(Range.findKey(windowRef(5), states)).toBeUndefined();
      expect(Range.findKey(staticRef(0, 1000), states)).toBeUndefined();
    });
  });

  describe("RECENT_REF", () => {
    it("should reference the rolling window of the recent range", () => {
      expect(Range.RECENT_REF).toEqual(windowRef(Number(TimeSpan.seconds(30))));
    });
  });

  describe("useResolveWindows", () => {
    it("should resolve each reference to its window in order", async () => {
      const range = await createTestRange(client);
      const { wrapper } = await createConsoleWrapper({ client });
      const refs = [windowRef(1000), keyRef(range.key), staticRef(0, 1000)];
      const { result } = renderHook(() => Range.useResolveWindows(refs), { wrapper });
      await waitFor(() =>
        expect(result.current).toEqual([
          dynamicWindow(1000),
          staticWindow(range.timeRange),
          staticWindow(new TimeRange(0, 1000)),
        ]),
      );
    });

    it("should leave out a key reference the Core does not hold", async () => {
      const { wrapper } = await createConsoleWrapper({ client });
      const refs = [keyRef(uuid.create()), windowRef(1000)];
      const { result } = renderHook(() => Range.useResolveWindows(refs), { wrapper });
      await waitFor(() => expect(result.current).toEqual([dynamicWindow(1000)]));
    });
  });
});
