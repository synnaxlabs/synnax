// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic, type Synnax } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  SyntheticCore,
} from "@synnaxlabs/client/testutil";
import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import type { FC, PropsWithChildren } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { z } from "zod";

import { Node } from "@/schematic/node";
import { telemTest } from "@/telem/aether/testutil";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";

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
) =>
  render(
    <Wrapper>
      <ReactFlowProvider>
        <Node.REGISTRY.symbol.Node
          nodeKey="s1"
          selected={false}
          onConfigChange={vi.fn()}
          config={schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config })}
        />
      </ReactFlowProvider>
    </Wrapper>,
  );

const createCustomSymbol = async (
  markup: string,
  data: Partial<z.input<typeof schematic.symbol.specZ>> = {},
  core?: SyntheticCore,
): Promise<{ key: string; client: Synnax; Wrapper: FC<PropsWithChildren> }> => {
  const client = createTestClient();
  const { key } = await client.schematics.symbols.create({
    name: "Custom",
    data: { drawing: { type: "svg", markup }, variant: "static", ...data },
  });
  const Wrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: symbolNode.REGISTRY,
    telemFactories: core == null ? undefined : [telemTest.createRemoteFactory(core)],
  });
  return { key, client, Wrapper };
};

describe("Symbol node", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("should fill a built-in symbol while a local state reads active", async () => {
    const { container } = renderNode({
      spec: { type: "builtin", name: "valve" },
      color: COLOR,
      states: [{ key: "active", source: { type: "local", value: 1 } }],
    });
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
  });

  it.each(["filter", "light"])(
    "should stroke the built-in %s, which has no stroke look, in the node color",
    async (name) => {
      const { container } = renderNode({
        spec: { type: "builtin", name },
        color: COLOR,
      });
      await vi.waitFor(() =>
        expect(container.querySelector("svg")?.getAttribute("style")).toContain(
          "--pluto-symbol-color: 171, 205, 239, 1",
        ),
      );
    },
  );

  it("should keep the look strokes of a custom symbol in a colored node", async () => {
    const { key, Wrapper } = await createCustomSymbol(
      '<svg xmlns="http://www.w3.org/2000/svg"><rect class="body" /></svg>',
      {
        parts: [{ key: "body", name: "Body", selectors: [".body"] }],
        looks: [
          {
            key: "red",
            part: "body",
            stroke: { type: "color", value: { type: "fixed", value: "#ff0000" } },
          },
        ],
      },
    );
    const { container } = renderNode(
      { spec: { type: "custom", key }, color: COLOR },
      Wrapper,
    );
    await vi.waitFor(() =>
      expect(container.querySelector<SVGElement>("rect.body")?.style.stroke).not.toBe(
        "",
      ),
    );
    expect(container.querySelector("svg")?.getAttribute("style")).not.toContain(
      "--pluto-symbol-color",
    );
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

  it("should show the stale and unknown indicators side by side", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "performance"] });
    const now = TimeStamp.now().sub(TimeSpan.minutes(1));
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const [reading] = core.data;
    const { findByRole } = renderNode(
      {
        spec: { type: "builtin", name: "valve" },
        states: [
          { key: "active", source: { type: "channel", key: reading.key } },
          // The synthetic Core has no such channel, so this state never gets data.
          { key: "fault", source: { type: "channel", key: 999_999 } },
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

  it("should show the unknown indicator for a local value that names no entry", async () => {
    const client = createTestClient();
    const { key: enumeration } = await client.enumerations.create({
      name: "Valve",
      entries: [
        { key: "closed", value: 0, name: "Closed" },
        { key: "open", value: 1, name: "Open" },
      ],
    });
    const Wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: symbolNode.REGISTRY,
    });
    const { findByRole } = renderNode(
      {
        spec: { type: "builtin", name: "valve" },
        states: [
          {
            key: "active",
            source: { type: "local", value: 5 },
            enumeration,
            match: { type: "exact" },
          },
        ],
      },
      Wrapper,
    );
    await findByRole("img", { name: "Unknown" });
  });

  it("should show no unknown indicator while the enumeration of a state is not loaded", async () => {
    const { container, queryByRole } = renderNode({
      spec: { type: "builtin", name: "valve" },
      states: [
        {
          key: "active",
          source: { type: "local", value: 5 },
          enumeration: "00000000-0000-0000-0000-000000000000",
          match: { type: "exact" },
        },
      ],
    });
    await vi.waitFor(() => expect(container.querySelector("path")).not.toBeNull());
    expect(queryByRole("img", { name: "Unknown" })).toBeNull();
  });

  it("should draw a custom symbol that the Core holds", async () => {
    const { key, Wrapper } = await createCustomSymbol(
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
        '<rect class="body" width="10" height="10" /></svg>',
    );
    const { container } = renderNode({ spec: { type: "custom", key } }, Wrapper);
    await vi.waitFor(() =>
      expect(container.querySelector(".pluto-symbol--custom rect.body")).not.toBeNull(),
    );
  });

  it("should not mark a built-in symbol as custom", async () => {
    const { container } = renderNode({ spec: { type: "builtin", name: "valve" } });
    await vi.waitFor(() =>
      expect(container.querySelector(".pluto-symbol svg")).not.toBeNull(),
    );
    expect(container.querySelector(".pluto-symbol--custom")).toBeNull();
  });

  it("should parse the markup of a custom symbol once while its channel reads", async () => {
    const now = TimeStamp.now();
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const [reading] = core.data;
    const { key, Wrapper } = await createCustomSymbol(
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
        '<rect class="body" width="10" height="10" /></svg>',
      {
        parts: [{ key: "body", name: "Body", selectors: [".body"] }],
        states: [
          {
            type: "enumeration",
            key: "active",
            name: "Active",
            match: { type: "exact" },
            default: 0,
          },
        ],
        looks: [
          {
            key: "on",
            part: "body",
            when: { type: "equals", state: "active", values: [1] },
            fill: { type: "color", value: { type: "fixed", value: "#ff0000" } },
          },
        ],
      },
      core,
    );
    const parse = vi.spyOn(DOMParser.prototype, "parseFromString");
    const { container } = renderNode(
      {
        spec: { type: "custom", key },
        states: [
          {
            key: "active",
            source: { type: "channel", key: reading.key },
            match: WITHIN,
          },
        ],
      },
      Wrapper,
    );
    await act(
      async () => await vi.waitFor(() => expect(core.streams(reading.key)).toBe(true)),
    );
    await vi.waitFor(() => expect(container.querySelector("rect.body")).not.toBeNull());
    const parsed = parse.mock.calls.length;
    await act(async () => core.advance(TimeSpan.seconds(1)));
    await vi.waitFor(() =>
      expect(container.querySelector("rect.body[fill='#ff0000']")).not.toBeNull(),
    );
    expect(parse.mock.calls.length).toBe(parsed);
  });

  it("should warn for a custom symbol whose markup does not parse", async () => {
    const { key, Wrapper } = await createCustomSymbol("<svg><rect></svg>");
    const { findByText } = renderNode({ spec: { type: "custom", key } }, Wrapper);
    await findByText("Invalid symbol");
  });

  it("should warn when a custom symbol fails to load", async () => {
    const { key, client } = await createCustomSymbol(
      '<svg xmlns="http://www.w3.org/2000/svg"><rect class="body" /></svg>',
    );
    // A user with no policies cannot read the symbol, so its load fails.
    const denied = await createTestClientWithPolicy(client, []);
    const Wrapper = await createAsyncSynnaxWrapper({
      client: denied,
      additionalRegistry: symbolNode.REGISTRY,
    });
    const { findByText } = renderNode({ spec: { type: "custom", key } }, Wrapper);
    await findByText("Failed to load symbol");
  });

  it("should draw its preview at the defaults of its states", async () => {
    const { container } = render(
      <LocalWrapper>
        <Node.REGISTRY.symbol.Preview
          {...schematic.symbolNodeConfigZ.parse({
            variant: "symbol",
            spec: { type: "builtin", name: "valve" },
            color: COLOR,
            states: [
              { key: "active", default: 1, source: { type: "local", value: 0 } },
            ],
          })}
        />
      </LocalWrapper>,
    );
    await vi.waitFor(() => expect(container.querySelector(FILLED)).not.toBeNull());
  });

  it.each([
    ["a built-in symbol that does not exist", { type: "builtin", name: "nope" }],
    ["a missing symbol", { type: "missing", key: "deleted" }],
  ] as const)("should warn for %s", async (_, spec) => {
    const { findByText } = renderNode({ spec });
    await findByText("Missing symbol");
  });
});
