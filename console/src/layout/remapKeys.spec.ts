// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { remapKeys } from "@/layout/remapKeys";
import { type SliceState, type State, ZERO_SLICE_STATE } from "@/layout/types";

const makeSlice = (
  layouts: Record<string, Partial<State>>,
  mosaic?: {
    activeTab?: string | null;
    tabs?: Array<{ tabKey: string; name: string }>;
  },
): SliceState => {
  const fullLayouts: Record<string, State> = {
    main: ZERO_SLICE_STATE.layouts.main,
  };
  for (const [key, partial] of Object.entries(layouts))
    fullLayouts[key] = {
      key,
      name: partial.name ?? key,
      type: partial.type ?? "lineplot",
      location: "mosaic",
      windowKey: "main",
      ...partial,
    } as State;

  const tabs =
    mosaic?.tabs ??
    Object.keys(layouts).map((k) => ({
      tabKey: k,
      name: fullLayouts[k].name,
      closable: true,
    }));

  return {
    ...ZERO_SLICE_STATE,
    layouts: fullLayouts,
    mosaics: {
      main: {
        activeTab: mosaic?.activeTab ?? Object.keys(layouts)[0] ?? null,
        focused: null,
        root: { key: 1, tabs, selected: tabs[0]?.tabKey },
      },
    },
  };
};

describe("remapKeys", () => {
  it("should assign new UUIDs to all non-main layout keys", () => {
    const slice = makeSlice({
      "old-key-1": { name: "Plot A", type: "lineplot" },
      "old-key-2": { name: "Schematic B", type: "schematic" },
    });

    const { slice: result, oldKeyForNew } = remapKeys(slice);

    expect(result.layouts).not.toHaveProperty("old-key-1");
    expect(result.layouts).not.toHaveProperty("old-key-2");
    expect(result.layouts).toHaveProperty("main");
    const newKeys = Object.keys(result.layouts).filter((k) => k !== "main");
    expect(newKeys).toHaveLength(2);
    for (const nk of newKeys) {
      expect(result.layouts[nk].key).toBe(nk);
      expect(oldKeyForNew.get(nk)).toBeDefined();
    }
  });

  it("should preserve the main layout unchanged", () => {
    const slice = makeSlice({ "old-key": { name: "Plot" } });
    const { slice: result } = remapKeys(slice);
    expect(result.layouts.main).toEqual(slice.layouts.main);
  });

  it("should remap mosaic tabKey references", () => {
    const slice = makeSlice({ abc: { name: "Plot" } });
    const { slice: result, oldKeyForNew } = remapKeys(slice);

    const newKey = Object.keys(result.layouts).find((k) => k !== "main")!;
    expect(oldKeyForNew.get(newKey)).toBe("abc");

    const tabs = result.mosaics.main.root.tabs!;
    expect(tabs).toHaveLength(1);
    expect(tabs[0].tabKey).toBe(newKey);
  });

  it("should remap mosaic activeTab", () => {
    const slice = makeSlice(
      { abc: { name: "Plot" } },
      { activeTab: "abc", tabs: [{ tabKey: "abc", name: "Plot" }] },
    );
    const { slice: result } = remapKeys(slice);

    const newKey = Object.keys(result.layouts).find((k) => k !== "main")!;
    expect(result.mosaics.main.activeTab).toBe(newKey);
  });

  it("should remap mosaic selected", () => {
    const slice = makeSlice({ abc: { name: "Plot" } });
    const { slice: result } = remapKeys(slice);

    const newKey = Object.keys(result.layouts).find((k) => k !== "main")!;
    expect(result.mosaics.main.root.selected).toBe(newKey);
  });

  it("should remap nested mosaic nodes", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
        aaa: {
          key: "aaa",
          name: "A",
          type: "lineplot",
          location: "mosaic",
          windowKey: "main",
        } as State,
        bbb: {
          key: "bbb",
          name: "B",
          type: "schematic",
          location: "mosaic",
          windowKey: "main",
        } as State,
      },
      mosaics: {
        main: {
          activeTab: "aaa",
          focused: null,
          root: {
            key: 1,
            direction: "x",
            first: { key: 2, tabs: [{ tabKey: "aaa", name: "A" }], selected: "aaa" },
            last: { key: 3, tabs: [{ tabKey: "bbb", name: "B" }], selected: "bbb" },
          },
        },
      },
    };

    const { slice: result, oldKeyForNew } = remapKeys(slice);
    const newA = [...oldKeyForNew.entries()].find(([, old]) => old === "aaa")![0];
    const newB = [...oldKeyForNew.entries()].find(([, old]) => old === "bbb")![0];

    expect(result.mosaics.main.root.first!.tabs![0].tabKey).toBe(newA);
    expect(result.mosaics.main.root.first!.selected).toBe(newA);
    expect(result.mosaics.main.root.last!.tabs![0].tabKey).toBe(newB);
    expect(result.mosaics.main.root.last!.selected).toBe(newB);
  });

  it("should produce different keys on each call", () => {
    const slice = makeSlice({ abc: { name: "Plot" } });
    const { slice: r1 } = remapKeys(slice);
    const { slice: r2 } = remapKeys(slice);

    const key1 = Object.keys(r1.layouts).find((k) => k !== "main")!;
    const key2 = Object.keys(r2.layouts).find((k) => k !== "main")!;
    expect(key1).not.toBe(key2);
  });

  it("should handle null activeTab", () => {
    const slice = makeSlice({ abc: { name: "Plot" } });
    slice.mosaics.main.activeTab = null;
    const { slice: result } = remapKeys(slice);
    expect(result.mosaics.main.activeTab).toBeNull();
  });

  it("should not modify the input slice", () => {
    const slice = makeSlice({ abc: { name: "Plot" } });
    const originalKeys = Object.keys(slice.layouts);
    remapKeys(slice);
    expect(Object.keys(slice.layouts)).toEqual(originalKeys);
    expect(slice.layouts.abc?.key).toBe("abc");
  });
});
