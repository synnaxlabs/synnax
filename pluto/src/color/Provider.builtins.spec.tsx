// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Color as Base } from "@synnaxlabs/lyra/color";
import { Theming } from "@synnaxlabs/lyra/theming";
import { box, color as xcolor, id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

import { Color } from "@/color";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { Gauge } from "@/vis/gauge";
import { gauge } from "@/vis/gauge/aether";
import { canvasTest } from "@/vis/render/testutil";

const client = createTestClient();

describe("Color.Provider", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    const Synnax = await createAsyncSynnaxWrapper({ client });
    wrapper = ({ children }) => (
      <Synnax>
        <Color.Provider>{children}</Color.Provider>
      </Synnax>
    );
    await waitForStreamLive(client.connection);
  });

  it("should offer a new series color in every picker", async () => {
    const { result } = renderHook(Base.usePresets, { wrapper });
    await waitFor(() => expect(result.current.length).toBeGreaterThan(0));
    const added = await client.colors.create({
      name: `series-${id.create()}`,
      light: "#1a2b3c",
      dark: "#1a2b3c",
      group: color.SERIES_PALETTE,
    });
    onTestFinished(async () => await client.colors.delete(added.key));
    await waitFor(() =>
      expect(result.current).toContainEqual(xcolor.construct("#1a2b3c")),
    );
  });

  it("should paint the series colors in the Aether components below", async () => {
    const recorder = canvasTest.record();
    const Synnax = await createAsyncSynnaxWrapper({
      client,
      renderContext: recorder,
      telemFactories: [],
      additionalRegistry: gauge.REGISTRY,
    });
    const strokes = (): string[] =>
      recorder.upper2d.calls
        .filter((c) => c.op === "set:strokeStyle")
        .map((c) => c.args[0] as string);
    const aetherKey = id.create();
    const { result } = renderHook(
      () => {
        Gauge.use({
          aetherKey,
          box: box.construct({ x: 0, y: 0 }, { width: 200, height: 200 }),
        });
        return Theming.use();
      },
      {
        wrapper: ({ children }) => (
          <Synnax>
            <Color.Provider>{children}</Color.Provider>
          </Synnax>
        ),
      },
    );
    const other = createTestClient();
    const [first] = (
      await other.colors.retrieve({ groups: [color.SERIES_PALETTE] })
    ).sort(color.compare);
    await waitFor(() =>
      expect(strokes()).toContain(xcolor.hex(Theming.resolve(first, result.current))),
    );
    const change = [
      color.set({ path: ["light"], value: "#2c3d4e" }),
      color.set({ path: ["dark"], value: "#2c3d4e" }),
    ];
    const { inverse } = color.reduceAll(first, change);
    await other.colors.dispatch(first.key, change);
    onTestFinished(async () => {
      await other.colors.dispatch(first.key, inverse);
    });
    await waitFor(() => expect(strokes().at(-1)).toBe("#2c3d4e"));
  });
});
