// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic } from "@synnaxlabs/client";
import { SyntheticCore } from "@synnaxlabs/client/testutil";
import type { Component } from "@synnaxlabs/lyra/component";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  act,
  render,
  type RenderResult,
  waitFor,
  within,
} from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import type { ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Setpoint } from "@/schematic/node/general/setpoint";
import { telemTest } from "@/telem/aether/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";
import { setpoint } from "@/vis/setpoint/aether";
import { Staleness } from "@/vis/staleness";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { value } from "@/vis/value/aether";

const NODE_KEY = "s1";

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: { ...value.REGISTRY, ...setpoint.REGISTRY },
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
            <Setpoint.spec.Node
              nodeKey={NODE_KEY}
              selected={false}
              onConfigChange={vi.fn()}
              config={schematic.setpointNodeConfigZ.parse({
                variant: "setpoint",
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

describe("Setpoint.Symbol", () => {
  it("should size the readout from the field's size", async () => {
    expect(await readoutTag("medium")).toBe("P");
    expect(await readoutTag("large")).toBe("H4");
  });
});

const staleColorHidden = vi.fn();

const StaleWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: {
    ...setpoint.REGISTRY,
    ...stalenessTest.createRegistry(value.Value.TYPE, value.Value.z, (s) =>
      staleColorHidden(s.staleColorHidden),
    ),
  },
  telemFactories: [new telemTest.TestFactory()],
  renderContext: canvasTest.record(),
});

const staleUI = (badged: boolean): ReactElement => (
  <StaleWrapper>
    <Staleness.Provider badged={badged}>
      <ReactFlowProvider>
        <Haul.Provider>
          <Setpoint.spec.Node
            nodeKey={NODE_KEY}
            selected={false}
            onConfigChange={vi.fn()}
            config={schematic.setpointNodeConfigZ.parse({
              variant: "setpoint",
              stateChannel: 7,
            })}
          />
        </Haul.Provider>
      </ReactFlowProvider>
    </Staleness.Provider>
  </StaleWrapper>
);

describe("Setpoint.Symbol staleness", () => {
  beforeEach(() => staleColorHidden.mockClear());
  afterEach(() => vi.useRealTimers());

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

  // The readback beside it reads a channel that sends, so the state channel is the only
  // difference.
  it("should never show the stale indicator with no state channel", async () => {
    // Fakes only the sweep's clock, so React and the worker still run on real timers.
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "performance"] });
    // Stamps every sample a minute ago, so a readback is stale once one arrives.
    const now = TimeStamp.now().sub(TimeSpan.minutes(1));
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const FeedWrapper = createSynnaxWrapper({
      client: null,
      additionalRegistry: { ...value.REGISTRY, ...setpoint.REGISTRY },
      telemFactories: [telemTest.createRemoteFactory(core)],
      renderContext: canvasTest.record(),
    });
    const node = (nodeKey: string, stateChannel: channel.Key): ReactElement => (
      <div data-testid={nodeKey}>
        <Setpoint.spec.Node
          nodeKey={nodeKey}
          selected={false}
          onConfigChange={vi.fn()}
          config={schematic.setpointNodeConfigZ.parse({
            variant: "setpoint",
            stateChannel,
          })}
        />
      </div>
    );
    const [reading] = core.data;
    const { getByTestId } = render(
      <FeedWrapper>
        <Staleness.Provider badged>
          <ReactFlowProvider>
            <Haul.Provider>
              {node("reading", reading.key)}
              {node("idle", 0)}
            </Haul.Provider>
          </ReactFlowProvider>
        </Staleness.Provider>
      </FeedWrapper>,
    );
    await act(
      async () => await vi.waitFor(() => expect(core.streams(reading.key)).toBe(true)),
    );
    await act(async () => core.advance(TimeSpan.seconds(1)));
    await within(getByTestId("reading")).findByRole("img", { name: "Stale" });
    await act(async () => {
      vi.advanceTimersByTime(TimeSpan.seconds(10).milliseconds);
    });
    expect(
      within(getByTestId("idle")).queryByRole("img", { name: "Stale" }),
    ).toBeNull();
  });
});
