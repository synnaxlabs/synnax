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
import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import type { FC, PropsWithChildren } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import type { NodeProps } from "@/schematic/node/spec";
import { symbolSpec } from "@/schematic/node/symbol/spec";
import type { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { control } from "@/telem/control/aether";
import { createSynnaxWrapper } from "@/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";

const chipState = z.object({ sink: z.unknown() }).loose();
const chipSinks = vi.fn();

// Stands in for the chip worker component and records the sink it acquires with.
class StubChip extends aether.Leaf<typeof chipState> {
  static readonly TYPE = control.Chip.TYPE;
  override schema = chipState;

  override afterUpdate(): void {
    chipSinks(this.state.sink);
  }
}

const indicatorState = z.object({}).loose();

class StubIndicator extends aether.Leaf<typeof indicatorState> {
  static readonly TYPE = control.Indicator.TYPE;
  override schema = indicatorState;
}

// Records each command write by its channel, in place of a control session.
class CommandFactory implements telem.Factory {
  type = "command";
  readonly sinks = new Map<channel.Key, telemTest.TestSink<number>>();

  create(spec: telem.Spec): telem.Telem | null {
    if (spec.type !== control.SetChannelValue.TYPE) return null;
    const sink = telemTest.sink<number>();
    this.sinks.set(control.setChannelValuePropsZ.parse(spec.props).channel, sink);
    return sink;
  }
}

const LocalWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: symbolNode.REGISTRY,
});

const createCoreWrapper = (core: SyntheticCore): FC<PropsWithChildren> =>
  createSynnaxWrapper({
    client: null,
    additionalRegistry: symbolNode.REGISTRY,
    telemFactories: [telemTest.createRemoteFactory(core)],
  });

// The light theme of the wrapper shows #abcdef as #a0c2e3.
const COLOR = { type: "fixed" as const, value: "#abcdef" };
const FILLED = "[fill='#a0c2e3']";

// Every sample of the synthetic Core reads as 1 through this match.
const WITHIN = { type: "within" as const, bounds: { lower: -2, upper: 2 } };

const renderNode = (
  config: Omit<z.input<typeof schematic.symbolNodeConfigZ>, "variant">,
  Wrapper: FC<PropsWithChildren> = LocalWrapper,
  onConfigChange: NodeProps["onConfigChange"] = vi.fn(),
) =>
  render(
    <Wrapper>
      <ReactFlowProvider>
        <symbolSpec.Node
          nodeKey="s1"
          selected={false}
          onConfigChange={onConfigChange}
          config={schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config })}
        />
      </ReactFlowProvider>
    </Wrapper>,
  );

describe("Symbol node", () => {
  afterEach(() => vi.useRealTimers());

  it("should fill a built-in symbol while a local state reads active", async () => {
    const { container } = renderNode({
      spec: { type: "builtin", name: "valve" },
      color: COLOR,
      states: [{ key: "active", source: { type: "local", value: 1 } }],
    });
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
  });

  it("should draw a state with no source at its default", async () => {
    const { container } = renderNode({
      spec: { type: "builtin", name: "valve" },
      color: COLOR,
      states: [{ key: "active", default: 1 }],
    });
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
  });

  it("should fill a built-in symbol while its channel reads active", async () => {
    const now = TimeStamp.now();
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const [reading] = core.data;
    const { container } = renderNode(
      {
        spec: { type: "builtin", name: "valve" },
        color: COLOR,
        states: [
          {
            key: "active",
            source: { type: "channel", key: reading.key },
            match: WITHIN,
          },
        ],
      },
      createCoreWrapper(core),
    );
    await act(
      async () => await vi.waitFor(() => expect(core.streams(reading.key)).toBe(true)),
    );
    expect(container.querySelector(FILLED)).toBeNull();
    await act(async () => core.advance(TimeSpan.seconds(1)));
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
  });

  it("should show the stale indicator while a state reads a stale channel", async () => {
    // Fakes only the sweep's clock, so React and the worker still run on real timers.
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "performance"] });
    // Stamps every sample a minute ago, so the reading is stale once one arrives.
    const now = TimeStamp.now().sub(TimeSpan.minutes(1));
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const [reading] = core.data;
    const { findByRole } = renderNode(
      {
        spec: { type: "builtin", name: "valve" },
        states: [
          {
            key: "active",
            source: { type: "channel", key: reading.key },
          },
        ],
      },
      createCoreWrapper(core),
    );
    await act(
      async () => await vi.waitFor(() => expect(core.streams(reading.key)).toBe(true)),
    );
    await act(async () => core.advance(TimeSpan.seconds(1)));
    await findByRole("img", { name: "Stale" });
  });

  it("should never show the stale indicator for a local state", async () => {
    const { container, queryByRole } = renderNode({
      spec: { type: "builtin", name: "valve" },
      color: COLOR,
      states: [{ key: "active", source: { type: "local", value: 1 } }],
    });
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
    expect(queryByRole("img", { name: "Stale" })).toBeNull();
  });

  it("should show the unknown indicator while a channel has no data", async () => {
    const now = TimeStamp.now();
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const [reading] = core.data;
    const { findByRole } = renderNode(
      {
        spec: { type: "builtin", name: "valve" },
        states: [
          {
            key: "active",
            source: { type: "channel", key: reading.key },
          },
        ],
      },
      createCoreWrapper(core),
    );
    await findByRole("img", { name: "Unknown" });
  });

  it("should show no unknown indicator for a state with no source", async () => {
    const { container, queryByRole } = renderNode({
      spec: { type: "builtin", name: "valve" },
      color: COLOR,
      states: [{ key: "active", default: 1 }],
    });
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
    expect(queryByRole("img", { name: "Unknown" })).toBeNull();
  });

  it("should show the stale and unknown indicators in one row", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "performance"] });
    const now = TimeStamp.now().sub(TimeSpan.minutes(1));
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const [reading] = core.data;
    const { findByRole } = renderNode(
      {
        spec: { type: "builtin", name: "valve" },
        // No enumeration has this key, so no value finds an entry.
        states: [
          {
            key: "active",
            source: { type: "channel", key: reading.key },
            enumeration: "00000000-0000-0000-0000-000000000000",
          },
        ],
      },
      createCoreWrapper(core),
    );
    await act(
      async () => await vi.waitFor(() => expect(core.streams(reading.key)).toBe(true)),
    );
    await act(async () => core.advance(TimeSpan.seconds(1)));
    const stale = await findByRole("img", { name: "Stale" });
    const unknown = await findByRole("img", { name: "Unknown" });
    expect(unknown.parentElement).toBe(stale.parentElement);
    expect(stale.closest(".pluto-grid__item")?.classList).toContain(
      "lyra--location-bottom",
    );
  });

  it.each([
    ["a built-in symbol that does not exist", { type: "builtin", name: "nope" }],
    ["a missing symbol", { type: "missing", key: "deleted" }],
  ] as const)("should warn for %s", async (_, spec) => {
    const { findByText } = renderNode({ spec });
    await findByText("Missing symbol");
  });
});

describe("Symbol node clicks", () => {
  const VALVE = { type: "builtin" as const, name: "valve" };
  const BUTTON = "button.pluto-symbol";

  it.each([
    ["open", 0, 1],
    ["close", 1, 0],
  ])("should %s a local valve on a click", (_, from, to) => {
    const onConfigChange = vi.fn();
    const { container } = renderNode(
      {
        spec: VALVE,
        states: [{ key: "active", source: { type: "local", value: from } }],
      },
      LocalWrapper,
      onConfigChange,
    );
    fireEvent.click(container.querySelector(BUTTON)!);
    expect(onConfigChange).toHaveBeenCalledWith({
      states: [expect.objectContaining({ source: { type: "local", value: to } })],
    });
  });

  it.each([
    [
      "while disabled",
      {
        disabled: true,
        states: [{ key: "active", source: { type: "local" as const, value: 0 } }],
      },
    ],
    ["for a state with no source", { states: [{ key: "active" }] }],
  ])("should take no clicks %s", (_, config) => {
    const { container } = renderNode({ spec: VALVE, ...config });
    expect(container.querySelector(BUTTON)).toBeNull();
    expect(container.querySelector(".pluto-symbol")).not.toBeNull();
  });

  it("should close a channel valve through its command channel", async () => {
    const now = TimeStamp.now();
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const [reading] = core.data;
    const commands = new CommandFactory();
    const Wrapper = createSynnaxWrapper({
      client: null,
      additionalRegistry: {
        ...symbolNode.REGISTRY,
        [StubChip.TYPE]: StubChip,
        [StubIndicator.TYPE]: StubIndicator,
      },
      telemFactories: [telemTest.createRemoteFactory(core), commands],
    });
    const { container } = renderNode(
      {
        spec: VALVE,
        color: COLOR,
        states: [
          {
            key: "active",
            source: { type: "channel", key: reading.key, command: 7 },
            match: WITHIN,
          },
        ],
      },
      Wrapper,
    );
    await act(
      async () => await vi.waitFor(() => expect(core.streams(reading.key)).toBe(true)),
    );
    expect(chipSinks).toHaveBeenCalledWith(
      control.acquireChannelControl({ channel: 7, authority: 255 }),
    );
    await act(async () => core.advance(TimeSpan.seconds(1)));
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
    fireEvent.click(container.querySelector(BUTTON)!);
    await vi.waitFor(() => expect(commands.sinks.get(7)?.values).toEqual([0]));
  });
});
