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
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Select } from "@/schematic/node/general/select";
import { telemTest } from "@/telem/aether/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";
import { setpoint } from "@/vis/setpoint/aether";
import { Staleness } from "@/vis/staleness";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { stateIndicator } from "@/vis/stateIndicator/aether";

const NODE_KEY = "c1";

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: { ...stateIndicator.REGISTRY, ...setpoint.REGISTRY },
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
            <Select.spec.Node
              nodeKey={NODE_KEY}
              selected={false}
              onConfigChange={vi.fn()}
              config={schematic.selectNodeConfigZ.parse({
                variant: "select",
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

describe("Select.Symbol", () => {
  it("should size the readout from the field's size", async () => {
    expect(await readoutTag("medium")).toBe("P");
    expect(await readoutTag("large")).toBe("H4");
  });
});

const StaleWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: {
    ...setpoint.REGISTRY,
    ...stalenessTest.createRegistry(
      stateIndicator.StateIndicator.TYPE,
      stateIndicator.stateZ,
    ),
  },
  telemFactories: [new telemTest.TestFactory()],
  renderContext: canvasTest.record(),
});

const renderStale = (badged: boolean) =>
  render(
    <StaleWrapper>
      <Staleness.Provider badged={badged}>
        <ReactFlowProvider>
          <Haul.Provider>
            <Select.spec.Node
              nodeKey={NODE_KEY}
              selected={false}
              onConfigChange={vi.fn()}
              config={schematic.selectNodeConfigZ.parse({
                variant: "select",
                stateChannel: 7,
              })}
            />
          </Haul.Provider>
        </ReactFlowProvider>
      </Staleness.Provider>
    </StaleWrapper>,
  );

const readout = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(
    `.${CSS.BE("symbol-field", "readout-value")}`,
  );
  if (el == null) throw new Error("expected a readout");
  return el;
};

describe("Select.Symbol staleness", () => {
  it("should show the stale indicator in place of the stale color", async () => {
    const { container, findByRole } = renderStale(true);
    const indicator = await findByRole("img", { name: "Stale" });
    expect(indicator.closest(".pluto-control-state")).not.toBeNull();
    expect(readout(container).style.color).toBe("");
  });

  it("should paint the stale color with no indicator when not badged", async () => {
    const { container, queryByRole } = renderStale(false);
    await waitFor(() => expect(readout(container).style.color).not.toBe(""));
    expect(queryByRole("img", { name: "Stale" })).toBeNull();
  });
});
