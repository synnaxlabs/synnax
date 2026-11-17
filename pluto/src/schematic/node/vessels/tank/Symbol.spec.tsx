// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { act, render, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { Border } from "@/schematic/node/common/border";
import { Tank } from "@/schematic/node/vessels/tank";

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
