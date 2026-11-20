// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@xyflow/react/dist/base.css";
import "@/schematic/Schematic.css";

import { type channel, schematic } from "@synnaxlabs/client";
import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Rate, scheduler, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { ReactFlowProvider } from "@xyflow/react";
import { type ReactElement } from "react";
import { describe, expect, it, onTestFinished, vi } from "vitest";
import { render } from "vitest-browser-react";

import { Input } from "@/schematic/node/general/input";
import { telemTest } from "@/telem/aether/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { input } from "@/vis/input/aether";
import { canvasTest } from "@/vis/render/testutil";
import { Staleness } from "@/vis/staleness";
import { stringValue } from "@/vis/stringValue/aether";

describe("Input.Symbol staleness", () => {
  // The readback beside it reads a channel that sends, so the state channel is the only
  // difference.
  it("should never show the stale indicator with no state channel", async () => {
    // Fakes only the sweep's clock, so React and the worker still run on real timers.
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "performance"] });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    // Stamps every sample a minute ago, so a readback is stale once one arrives.
    const now = TimeStamp.now().sub(TimeSpan.minutes(1));
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const Wrapper = createSynnaxWrapper({
      client: null,
      additionalRegistry: { ...stringValue.REGISTRY, ...input.REGISTRY },
      telemFactories: [telemTest.createRemoteFactory(core)],
      renderContext: canvasTest.record(),
    });
    const node = (nodeKey: string, stateChannel: channel.Key): ReactElement => (
      <div data-testid={nodeKey}>
        <Input.spec.Node
          nodeKey={nodeKey}
          selected={false}
          onConfigChange={vi.fn()}
          config={schematic.inputNodeConfigZ.parse({ variant: "input", stateChannel })}
        />
      </div>
    );
    const [reading] = core.data;
    const screen = await render(
      <Wrapper>
        <Staleness.Provider badged>
          <ReactFlowProvider>
            <Haul.Provider>
              {node("reading", reading.key)}
              {node("idle", 0)}
            </Haul.Provider>
          </ReactFlowProvider>
        </Staleness.Provider>
      </Wrapper>,
    );
    await vi.waitFor(() => expect(core.streams(reading.key)).toBe(true));
    core.advance(TimeSpan.seconds(1));
    await expect
      .element(screen.getByTestId("reading").getByRole("img", { name: "Stale" }))
      .toBeInTheDocument();
    vi.advanceTimersByTime(TimeSpan.seconds(10).milliseconds);
    await scheduler.nextFrame();
    expect(
      screen.getByTestId("idle").getByRole("img", { name: "Stale" }).query(),
    ).toBeNull();
  });
});
