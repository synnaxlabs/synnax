// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import {
  color,
  dynamicWindow,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  uuid,
} from "@synnaxlabs/x";
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

  describe("keyOf", () => {
    it("should key a saved range by its key", () => {
      expect(Range.keyOf(keyRef("a"))).toBe("a");
    });

    it("should key a window by its span or its bounds", () => {
      expect(Range.keyOf(windowRef(1000))).toBe("window:1000");
      expect(Range.keyOf(staticRef(0, 1000))).toBe("window:0:1000");
    });
  });

  describe("useResolveRanges", () => {
    const RED = color.construct("#ff0000");
    const BLUE = color.construct("#0000ff");
    const GREEN = color.construct("#00ff00");

    it("should zero a saved range at its start, or at its T0", async () => {
      const range = await createTestRange(client);
      const t0 = range.timeRange.start.add(TimeSpan.seconds(2));
      const { wrapper } = await createConsoleWrapper({ client });
      const refs: ranger.Ref[] = [
        keyRef(range.key),
        { variant: "key", key: range.key, t0 },
      ];
      const { result } = renderHook(() => Range.useResolveRanges(refs, [RED, BLUE]), {
        wrapper,
      });
      const spec = {
        key: range.key,
        window: staticWindow(range.timeRange),
        label: range.name,
      };
      await waitFor(() =>
        expect(result.current).toEqual([
          { ...spec, color: RED, zero: range.timeRange.start, customZero: false },
          { ...spec, color: BLUE, zero: t0, customZero: true },
        ]),
      );
    });

    it("should zero a fixed window at its start and give a rolling window no zero", async () => {
      const { wrapper } = await createConsoleWrapper({ client });
      const span = TimeSpan.minutes(5).valueOf();
      const refs = [staticRef(1000, 2000), windowRef(Number(span))];
      const { result } = renderHook(() => Range.useResolveRanges(refs, [RED, BLUE]), {
        wrapper,
      });
      expect(result.current).toEqual([
        expect.objectContaining({
          key: "window:1000:2000",
          color: RED,
          zero: new TimeStamp(1000),
        }),
        {
          key: `window:${span}`,
          window: dynamicWindow(span),
          label: "Rolling 5min",
          color: BLUE,
          zero: undefined,
        },
      ]);
    });

    it("should leave out a saved range the Core does not hold", async () => {
      const { wrapper } = await createConsoleWrapper({ client });
      const refs = [keyRef(uuid.create()), windowRef(1000)];
      const { result } = renderHook(() => Range.useResolveRanges(refs, [RED, GREEN]), {
        wrapper,
      });
      await waitFor(() =>
        expect(result.current).toEqual([expect.objectContaining({ color: GREEN })]),
      );
    });
  });
});
