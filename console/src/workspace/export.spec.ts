// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type SliceState, type State, ZERO_SLICE_STATE } from "@/layout/types";
import { purgeExcludedLayouts } from "@/workspace/purgeExcludedLayouts";

import { deduplicateLayoutNames, removeDirectory } from "./export";

describe("removeDirectory", () => {
  it("should replace forward slashes with underscores", () => {
    expect(removeDirectory("path/to/name")).toBe("path_to_name");
  });

  it("should replace backslashes with underscores", () => {
    expect(removeDirectory("path\\to\\name")).toBe("path_to_name");
  });

  it("should replace mixed slashes", () => {
    expect(removeDirectory("path/to\\name")).toBe("path_to_name");
  });

  it("should return the same string when no slashes are present", () => {
    expect(removeDirectory("simple-name")).toBe("simple-name");
  });
});

const makeLayout = (overrides: Partial<State>): State =>
  ({
    key: "test",
    name: "Test",
    type: "lineplot",
    location: "mosaic",
    windowKey: "main",
    ...overrides,
  }) as State;

describe("deduplicateLayoutNames", () => {
  it("should deduplicate layouts with the same name", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
        a: makeLayout({ key: "a", name: "Plot" }),
        b: makeLayout({ key: "b", name: "Plot" }),
      },
    };
    deduplicateLayoutNames(slice);
    const names = Object.values(slice.layouts)
      .filter((l) => l.key !== "main")
      .map((l) => l.name);
    expect(new Set(names).size).toBe(names.length);
    expect(names).toContain("Plot");
  });

  it("should sanitize slashes in layout names", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
        a: makeLayout({ key: "a", name: "path/to/plot" }),
      },
    };
    deduplicateLayoutNames(slice);
    expect(slice.layouts.a.name).toBe("path_to_plot");
  });

  it("should handle layouts with no duplicates", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
        a: makeLayout({ key: "a", name: "Plot A" }),
        b: makeLayout({ key: "b", name: "Plot B" }),
      },
    };
    deduplicateLayoutNames(slice);
    expect(slice.layouts.a.name).toBe("Plot A");
    expect(slice.layouts.b.name).toBe("Plot B");
  });
});

describe("purgeExcludedLayouts", () => {
  it("should remove layouts with excludeFromWorkspace set to true", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
        keep: makeLayout({ key: "keep", name: "Keep" }),
        exclude: makeLayout({
          key: "exclude",
          name: "Exclude",
          excludeFromWorkspace: true,
        }),
      },
    };
    const result = purgeExcludedLayouts(slice);
    expect(result.layouts).toHaveProperty("keep");
    expect(result.layouts).not.toHaveProperty("exclude");
  });

  it("should remove layouts with location set to modal", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
        keep: makeLayout({ key: "keep", name: "Keep" }),
        modal: makeLayout({ key: "modal", name: "Modal", location: "modal" }),
      },
    };
    const result = purgeExcludedLayouts(slice);
    expect(result.layouts).toHaveProperty("keep");
    expect(result.layouts).not.toHaveProperty("modal");
  });

  it("should not mutate the original slice state", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
        keep: makeLayout({ key: "keep", name: "Keep" }),
        exclude: makeLayout({
          key: "exclude",
          name: "Exclude",
          excludeFromWorkspace: true,
        }),
      },
    };
    purgeExcludedLayouts(slice);
    expect(slice.layouts).toHaveProperty("exclude");
  });

  it("should preserve the main layout", () => {
    const slice: SliceState = {
      ...ZERO_SLICE_STATE,
      layouts: {
        main: ZERO_SLICE_STATE.layouts.main,
      },
    };
    const result = purgeExcludedLayouts(slice);
    expect(result.layouts).toHaveProperty("main");
  });
});
