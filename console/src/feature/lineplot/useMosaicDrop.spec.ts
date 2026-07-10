// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot as clientLineplot } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { client, project } from "@/feature/lineplot/testutil";
import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

describe("LinePlot.useMosaicDrop", () => {
  it("places the plot into the target mosaic node", async () => {
    const plot = await client.lineplots.create(await project(), {
      name: uniqueName("plot"),
    });
    const { result, store } = await renderHookWithConsole(
      () => LinePlot.useMosaicDrop(),
      { client },
    );
    result.current({
      id: clientLineplot.ontologyID(plot.key),
      nodeKey: 3,
      location: "top",
    });
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), plot.key)?.name).toBe(plot.name),
    );
    const layout = Session.Layout.select(store.getState(), plot.key);
    expect(layout?.location).toBe("mosaic");
    expect(layout?.tab).toMatchObject({ mosaicKey: 3, location: "top" });
  });
});
