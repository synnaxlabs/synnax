// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, color, schematic } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { color as xcolor, uuid } from "@synnaxlabs/x";
import { act, render, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { Border } from "@/schematic/node/common/border";
import { Tank } from "@/schematic/node/vessels/tank";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const state = z.object({ cornerRadii: z.unknown() }).loose();

const updated = vi.fn();

// Stands in for the scale worker component that draws the tank's fill.
class StubScale extends aether.Leaf<typeof state> {
  static readonly TYPE = "scale";
  schema = state;

  afterUpdate(): void {
    updated(this.state.cornerRadii);
  }
}

const renderTank = async (
  channel: channel.Key,
  borderRadius?: schematic.TankNodeConfig["borderRadius"],
): Promise<void> => {
  const [workerSide, mainSide] = aether.createMockPair();
  aether.render({ worker: workerSide, registry: { [StubScale.TYPE]: StubScale } });
  await act(async () => {
    render(
      <Aether.Provider worker={mainSide}>
        <ReactFlowProvider>
          <Haul.Provider>
            <div data-id="t1">
              <Tank.spec.Node
                nodeKey="t1"
                selected={false}
                onConfigChange={vi.fn()}
                config={schematic.tankNodeConfigZ.parse({
                  variant: "tank",
                  channel,
                  borderRadius,
                })}
              />
            </div>
          </Haul.Provider>
        </ReactFlowProvider>
      </Aether.Provider>,
    );
  });
};

describe("Tank.Symbol", () => {
  beforeEach(() => {
    updated.mockClear();
  });

  it("should draw no fill when the tank reads no channel", async () => {
    await renderTank(0);
    expect(updated).not.toHaveBeenCalled();
  });

  it("should draw the fill when the tank reads a channel", async () => {
    await renderTank(7);
    await waitFor(() => expect(updated).toHaveBeenCalled());
  });

  it("should clip the fill to the default radius when the radius is absent", async () => {
    await renderTank(7, Border.DEFAULT_RADIUS);
    await waitFor(() => expect(updated).toHaveBeenCalled());
    const withDefault = updated.mock.lastCall?.[0];
    updated.mockClear();
    await renderTank(7);
    await waitFor(() => expect(updated).toHaveBeenCalled());
    expect(withDefault).toBeDefined();
    expect(updated.mock.lastCall?.[0]).toEqual(withDefault);
  });
});

describe("Tank.Symbol named colors", () => {
  const renderNamed = async (
    config: Partial<schematic.TankNodeConfig>,
  ): Promise<HTMLElement> => {
    const client = createTestClient();
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const { container } = render(
      <ReactFlowProvider>
        <Haul.Provider>
          <div data-id="t1">
            <Tank.spec.Node
              nodeKey="t1"
              selected={false}
              onConfigChange={vi.fn()}
              config={schematic.tankNodeConfigZ.parse({ variant: "tank", ...config })}
            />
          </div>
        </Haul.Provider>
      </ReactFlowProvider>,
      { wrapper },
    );
    const tank = container.querySelector<HTMLElement>(".pluto-tank");
    if (tank == null) throw new Error("expected a tank element");
    return tank;
  };
  const stroke = (tank: HTMLElement): string =>
    tank.style.getPropertyValue("--pluto-symbol-color");

  it("should paint and repaint its named stroke and fill", async () => {
    const client = createTestClient();
    const [s, f] = await client.colors.create([
      { name: `stroke-${uuid.create()}`, light: "#123456" },
      { name: `fill-${uuid.create()}`, light: "#654321" },
    ]);
    const tank = await renderNamed({
      strokeColor: { type: "named", key: s.key },
      fillColor: { type: "named", key: f.key },
    });
    await waitFor(() => expect(stroke(tank)).toBe("18, 52, 86, 1"));
    expect(tank.style.backgroundColor).toBe("rgb(101, 67, 33)");
    await client.colors.dispatch(
      s.key,
      color.set({ path: ["light"], value: xcolor.construct("#abcdef") }),
    );
    await client.colors.dispatch(
      f.key,
      color.set({ path: ["light"], value: xcolor.construct("#fedcba") }),
    );
    await waitFor(() => expect(stroke(tank)).toBe("171, 205, 239, 1"));
    await waitFor(() => expect(tank.style.backgroundColor).toBe("rgb(254, 220, 186)"));
  });

  it("should paint no fill for a named color the Core does not hold", async () => {
    const client = createTestClient();
    const s = await client.colors.create({
      name: `stroke-${uuid.create()}`,
      light: "#123456",
    });
    const tank = await renderNamed({
      strokeColor: { type: "named", key: s.key },
      fillColor: { type: "named", key: uuid.create() },
    });
    // The stroke resolving shows that the color library has loaded.
    await waitFor(() => expect(stroke(tank)).toBe("18, 52, 86, 1"));
    expect(tank.style.backgroundColor).toBe("");
  });
});
