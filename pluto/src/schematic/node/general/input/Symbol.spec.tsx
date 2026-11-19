// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type Component } from "@synnaxlabs/lyra/component";
import { Haul } from "@synnaxlabs/lyra/haul";
import { act, render, type RenderResult, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";
import { Input } from "@/schematic/node/general/input";
import { telemTest } from "@/telem/aether/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { input } from "@/vis/input/aether";
import { canvasTest } from "@/vis/render/testutil";
import { Staleness } from "@/vis/staleness";
import { stringValue } from "@/vis/stringValue/aether";

const NODE_KEY = "i1";

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: { ...stringValue.REGISTRY, ...input.REGISTRY },
  telemFactories: [new telemTest.TestFactory()],
  renderContext: canvasTest.record(),
});

const readoutTag = async (size: Component.Size): Promise<string | undefined> => {
  let result: RenderResult | undefined;
  await act(async () => {
    result = render(
      <Wrapper>
        <ReactFlowProvider>
          <Haul.Provider>
            <Input.spec.Node
              nodeKey={NODE_KEY}
              selected={false}
              onConfigChange={vi.fn()}
              config={schematic.inputNodeConfigZ.parse({
                variant: "input",
                stateChannel: 7,
                size,
              })}
            />
          </Haul.Provider>
        </ReactFlowProvider>
      </Wrapper>,
    );
  });
  return result?.container.querySelector(`.${CSS.BE("symbol-field", "readout-value")}`)
    ?.tagName;
};

describe("Input.Symbol", () => {
  it("should size the readout from the field's size", async () => {
    expect(await readoutTag("medium")).toBe("P");
    expect(await readoutTag("large")).toBe("H4");
  });
});

const readbackState = z
  .object({
    stale: z.boolean().default(false),
    staleColorHidden: z.boolean().default(false),
  })
  .loose();

const staleColorHidden = vi.fn();

// Stands in for the readback worker component and reports its source as stale.
class StaleReadback extends aether.Leaf<typeof readbackState> {
  static readonly TYPE = stringValue.StringValue.TYPE;
  schema = readbackState;

  afterUpdate(): void {
    staleColorHidden(this.state.staleColorHidden);
    if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
  }
}

const StaleWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: { ...input.REGISTRY, [StaleReadback.TYPE]: StaleReadback },
  telemFactories: [new telemTest.TestFactory()],
  renderContext: canvasTest.record(),
});

const staleUI = (badged: boolean): ReactElement => (
  <StaleWrapper>
    <Staleness.Provider badged={badged}>
      <ReactFlowProvider>
        <Haul.Provider>
          <Input.spec.Node
            nodeKey={NODE_KEY}
            selected={false}
            onConfigChange={vi.fn()}
            config={schematic.inputNodeConfigZ.parse({
              variant: "input",
              stateChannel: 7,
            })}
          />
        </Haul.Provider>
      </ReactFlowProvider>
    </Staleness.Provider>
  </StaleWrapper>
);

describe("Input.Symbol staleness", () => {
  beforeEach(() => staleColorHidden.mockClear());

  it("should show the stale indicator in place of the stale color", async () => {
    const { findByRole } = render(staleUI(true));
    const indicator = await findByRole("img", { name: "Stale" });
    expect(indicator.closest(".pluto-control-state")).not.toBeNull();
    expect(staleColorHidden).toHaveBeenLastCalledWith(true);
  });

  // Starts badged, so the indicator proves the stale state reached the symbol first.
  it("should paint the stale color with no indicator when not badged", async () => {
    const { findByRole, queryByRole, rerender } = render(staleUI(true));
    await findByRole("img", { name: "Stale" });
    rerender(staleUI(false));
    await waitFor(() => expect(staleColorHidden).toHaveBeenLastCalledWith(false));
    expect(queryByRole("img", { name: "Stale" })).toBeNull();
  });
});
