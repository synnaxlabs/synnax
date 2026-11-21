// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { render, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { StateIndicator } from "@/schematic/node/general/stateIndicator";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { Staleness } from "@/vis/staleness";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { stateIndicator } from "@/vis/stateIndicator/aether";

// React Flow resizes a node by pointer drags that jsdom cannot run, so the spec calls
// the resize callback that the control receives.
const mocks = vi.hoisted(() => ({
  onResize: [] as Array<(e: unknown, d: { width: number; height: number }) => void>,
}));

vi.mock("@xyflow/react", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    NodeResizeControl: ({
      onResize,
    }: {
      onResize: (e: unknown, d: { width: number; height: number }) => void;
    }) => {
      mocks.onResize.push(onResize);
      return null;
    },
  };
});

const state = z.object({}).loose();

// Stands in for the state indicator worker component.
class StubStateIndicator extends aether.Leaf<typeof state> {
  static readonly TYPE = "StateIndicator";
  override schema = state;
}

describe("StateIndicator.Symbol", () => {
  beforeEach(() => {
    mocks.onResize = [];
  });

  it("should round the inline size of a resize to whole pixels", () => {
    const [workerSide, mainSide] = aether.createMockPair();
    aether.render({
      worker: workerSide,
      registry: { [StubStateIndicator.TYPE]: StubStateIndicator },
    });
    const onConfigChange = vi.fn();
    render(
      <Aether.Provider worker={mainSide}>
        <ReactFlowProvider>
          <Haul.Provider>
            <div data-id="s1">
              <StateIndicator.spec.Node
                nodeKey="s1"
                selected
                onConfigChange={onConfigChange}
                config={schematic.stateIndicatorNodeConfigZ.parse({
                  variant: "state_indicator",
                })}
              />
            </div>
          </Haul.Provider>
        </ReactFlowProvider>
      </Aether.Provider>,
    );
    expect(mocks.onResize.length).toBeGreaterThan(0);
    mocks.onResize[0]({}, { width: 120.6, height: 30 });
    expect(onConfigChange).toHaveBeenLastCalledWith({ inlineSize: 121 });
  });
});

const StaleWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: stalenessTest.createRegistry(
    stateIndicator.StateIndicator.TYPE,
    stateIndicator.stateZ,
  ),
  telemFactories: [],
});

const renderStale = (badged: boolean) =>
  render(
    <StaleWrapper>
      <Staleness.Provider badged={badged}>
        <ReactFlowProvider>
          <StateIndicator.spec.Node
            nodeKey="s1"
            selected={false}
            onConfigChange={vi.fn()}
            config={schematic.stateIndicatorNodeConfigZ.parse({
              variant: "state_indicator",
              channel: 1,
            })}
          />
        </ReactFlowProvider>
      </Staleness.Provider>
    </StaleWrapper>,
  );

describe("StateIndicator.Symbol staleness", () => {
  it("should show the stale indicator in place of the stale color", async () => {
    const { findByRole, getByText } = renderStale(true);
    const indicator = await findByRole("img", { name: "Stale" });
    expect(indicator.closest(".pluto-grid__item")?.classList).toContain(
      "lyra--location-bottom",
    );
    expect(getByText("Unknown").style.color).toBe("");
  });

  it("should paint the stale color with no indicator when not badged", async () => {
    const { getByText, queryByRole } = renderStale(false);
    await waitFor(() => expect(getByText("Unknown").style.color).not.toBe(""));
    expect(queryByRole("img", { name: "Stale" })).toBeNull();
  });
});
