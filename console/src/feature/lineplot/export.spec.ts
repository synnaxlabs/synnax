// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { client, createPreloadedState, project } from "@/feature/lineplot/testutil";
import { createTestStore, uniqueName } from "@/testutil";

const extract = LinePlot.EXTRACTORS[LinePlot.LAYOUT_TYPE];

describe("lineplot export", () => {
  it("serializes the plot with its layout type and the layout's name", async () => {
    const plot = await client.lineplots.create(await project(), {
      name: uniqueName("plot"),
    });
    const store = await createTestStore({
      preloadedState: createPreloadedState(plot.key, "Renamed Layout"),
    });
    const file = await extract(plot.key, { store, client });
    expect(file.name).toBe("Renamed Layout");
    const data = JSON.parse(file.data);
    expect(data.type).toBe(LinePlot.LAYOUT_TYPE);
    expect(data.key).toBe(plot.key);
    expect(data.name).toBe(plot.name);
  });

  it("falls back to the server name when no layout exists", async () => {
    const plot = await client.lineplots.create(await project(), {
      name: uniqueName("plot"),
    });
    const store = await createTestStore();
    const file = await extract(plot.key, { store, client });
    expect(file.name).toBe(plot.name);
  });

  it("throws a DisconnectedError without a client", async () => {
    const store = await createTestStore();
    await expect(extract("some-key", { store, client: null })).rejects.toSatisfy((e) =>
      DisconnectedError.matches(e),
    );
  });
});
