// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/platform/lineplot";
import { Session } from "@/session";
import { createTestStore } from "@/testutil";

const run = async (arg?: LinePlot.CreateArg) => {
  const store = await createTestStore();
  const creator = LinePlot.create(arg);
  const layout = creator({ dispatch: store.dispatch, store, windowKey: "main" });
  return { store, layout };
};

describe("LinePlot.create", () => {
  it("returns a mosaic line plot layout with defaults and registers the plot", async () => {
    const { store, layout } = await run();
    expect(layout.type).toBe(LinePlot.LAYOUT_TYPE);
    expect(layout.name).toBe("Line Plot");
    expect(layout.location).toBe("mosaic");
    expect(layout.icon).toBe("Visualize");
    expect(layout.key).toBeDefined();
    expect(
      Session.LinePlot.selectSliceState(store.getState()).plots[layout.key as string],
    ).toBeDefined();
  });

  it("honors name, location, window, and tab overrides", async () => {
    const { layout } = await run({
      name: "Custom Plot",
      location: "window",
      window: { title: "W" },
      tab: { closable: false },
    });
    expect(layout.name).toBe("Custom Plot");
    expect(layout.location).toBe("window");
    expect(layout.window).toEqual({ title: "W" });
    expect(layout.tab).toEqual({ closable: false });
  });

  it("uses a valid provided key verbatim", async () => {
    const key = uuid.create();
    const { layout } = await run({ key });
    expect(layout.key).toBe(key);
  });

  it("generates a fresh key when the provided key is not a valid line plot key", async () => {
    const { layout } = await run({ key: "not-a-uuid" });
    expect(layout.key).not.toBe("not-a-uuid");
    expect(lineplot.keyZ.safeParse(layout.key).success).toBe(true);
  });
});
