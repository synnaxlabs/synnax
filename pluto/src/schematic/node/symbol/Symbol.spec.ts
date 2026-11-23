// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  COLOR,
  CommandFactory,
  createControlStubs,
  createWrapper,
  FILLED,
  renderNode,
  WITHIN,
} from "@/schematic/node/symbol/testutil";
import { control } from "@/telem/control/aether";

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
      createWrapper({ core }),
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
      createWrapper({ core }),
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
      createWrapper({ core }),
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
      createWrapper({ core }),
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
      undefined,
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
    const { registry, chipSinks } = createControlStubs();
    const Wrapper = createWrapper({ core, commands, registry });
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
