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
} from "@synnaxlabs/client/testutil";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { ReactFlowProvider } from "@xyflow/react";
import type { FC, PropsWithChildren } from "react";
import { assert, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";
import type { z } from "zod";

import { Node } from "@/schematic/node";
import { queryButton } from "@/schematic/node/common/toggle/testutil";
import { createChannelTelem } from "@/schematic/node/testutil";
import type { telem } from "@/telem/aether";
import { controlTest } from "@/telem/control/aether/testutil";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";

// A node with a command channel mounts the control chip.
const REGISTRY = { ...symbolNode.REGISTRY, ...controlTest.createRegistry() };

const MARKUP =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
  '<rect class="lamp" width="10" height="10" /></svg>';

// The lamp blinks while the active state reads 1.
const DATA: Partial<schematic.symbol.Spec> = {
  parts: [{ key: "lamp", name: "Lamp", selectors: [".lamp"] }],
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
      key: "blink",
      part: "lamp",
      blink: { period: 1e9 },
      when: { type: "equals", state: "active", values: [1] },
    },
  ],
};

type Config = Omit<z.input<typeof schematic.symbolNodeConfigZ>, "variant">;

const renderNode = async (
  Wrapper: FC<PropsWithChildren>,
  config: Config,
  onConfigChange = vi.fn(),
) =>
  await render(
    <Wrapper>
      <Node.Custom.MotionProvider>
        <ReactFlowProvider>
          <Node.REGISTRY.symbol.Node
            nodeKey="s1"
            selected={false}
            onConfigChange={onConfigChange}
            config={schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config })}
          />
        </ReactFlowProvider>
      </Node.Custom.MotionProvider>
    </Wrapper>,
  );

const renderLamp = async (config: Pick<Config, "motionDisabled"> = {}) => {
  const client = createTestClient();
  const symbol = await client.schematics.symbols.create({
    name: "Lamp",
    data: { drawing: { type: "svg", markup: MARKUP }, variant: "static", ...DATA },
  });
  const Wrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: REGISTRY,
  });
  const { container } = await renderNode(Wrapper, {
    spec: { type: "custom", key: symbol.key },
    states: [{ key: "active", source: { type: "local", value: 1 } }],
    ...config,
  });
  return await vi.waitUntil(() => container.querySelector(".lamp"));
};

const BuiltInWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: symbolNode.REGISTRY,
});

// The theme gives the colors that a drawing with no node color falls back to.
const renderSolid = async (name: string, color?: string): Promise<Element> => {
  const { container } = await renderThemed(
    <BuiltInWrapper>
      <ReactFlowProvider>
        <Node.REGISTRY.symbol.Node
          nodeKey="s1"
          selected={false}
          onConfigChange={vi.fn()}
          config={schematic.symbolNodeConfigZ.parse({
            variant: "symbol",
            spec: { type: "builtin", name },
            color: color == null ? undefined : { type: "fixed", value: color },
          })}
        />
      </ReactFlowProvider>
    </BuiltInWrapper>,
  );
  return await vi.waitUntil(() => container.querySelector(".pluto--solid"));
};

// renderValve renders a valve whose active state has source, and returns its button.
const renderValve = async (
  factory: telem.Factory,
  source: schematic.StateBinding["source"],
): Promise<HTMLButtonElement> => {
  const Wrapper = createSynnaxWrapper({
    client: null,
    additionalRegistry: REGISTRY,
    telemFactories: [factory],
  });
  const { container } = await renderNode(Wrapper, {
    spec: { type: "builtin", name: "valve" },
    states: [{ key: "active", source }],
  });
  return await vi.waitUntil(() => queryButton(container));
};

// createAsk creates a symbol whose click asks for a value of state.
const createAsk = async (client: Synnax, state: schematic.symbol.State) =>
  await client.schematics.symbols.create({
    name: "Ask",
    data: {
      drawing: { type: "svg", markup: MARKUP },
      variant: "static",
      states: [state],
      commands: [{ key: "set", name: "Set", state: state.key, value: { type: "ask" } }],
      interactions: [{ type: "fire", key: "click", command: "set" }],
    },
  });

// renderAsk renders a node of symbol, whose state reads a local value of 0, and clicks
// it. It returns the spy on its config changes and its button.
const renderAsk = async (client: Synnax, symbol: schematic.symbol.Symbol) => {
  const Wrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: REGISTRY,
  });
  const onConfigChange = vi.fn();
  const { container } = await renderNode(
    Wrapper,
    {
      spec: { type: "custom", key: symbol.key },
      states: [{ key: symbol.data.states[0].key, source: { type: "local", value: 0 } }],
    },
    onConfigChange,
  );
  const button = await vi.waitUntil(() => queryButton(container));
  await page.elementLocator(button).click();
  return { onConfigChange, button };
};

describe("Symbol node", () => {
  it("should move a part by the value of its bound state", async () => {
    const lamp = await renderLamp();
    // The lamp dims for the second half of each one second period.
    await vi.waitFor(() => expect(lamp.getAttribute("opacity")).toBe("0.2"), {
      timeout: 2000,
    });
  });

  it("should not move a part on a node with motion disabled", async () => {
    const lamp = await renderLamp({ motionDisabled: true });
    for (let i = 0; i < 3; i++) await new Promise(requestAnimationFrame);
    expect(lamp.getAttribute("opacity")).toBeNull();
  });

  it("should keep the stroke width of a custom drawing", async () => {
    const lamp = await renderLamp();
    expect(getComputedStyle(lamp).strokeWidth).toBe("1px");
  });

  it("should draw a copy of a built-in as wide as the built-in", async () => {
    const client = createTestClient();
    const data = Node.BUILT_INS.get("ball_valve");
    assert(data != null);
    const copy = await client.schematics.symbols.create({ name: "Ball (copy)", data });
    const Wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: REGISTRY,
    });
    const widthOf = async (spec: Config["spec"]) => {
      const { container } = await renderNode(Wrapper, { spec });
      const circle = await vi.waitUntil(() => container.querySelector("circle"));
      return getComputedStyle(circle).strokeWidth;
    };
    expect([
      await widthOf({ type: "custom", key: copy.key }),
      await widthOf({ type: "builtin", name: "ball_valve" }),
    ]).toEqual(["2px", "2px"]);
  });

  it("should ask which command to send while its state is unknown", async () => {
    const { sinks, factory } = createChannelTelem();
    const button = await renderValve(factory, { type: "channel", key: 5, command: 7 });
    await expect.element(page.getByRole("img", { name: "Unknown" })).toBeVisible();
    await page.elementLocator(button).click();
    const open = page.getByRole("menuitem", { name: "Open" });
    await expect.element(page.getByRole("menuitem", { name: "Close" })).toBeVisible();
    await open.click();
    await expect.poll(() => sinks.get(7)?.values).toEqual([1]);
    await expect.element(open).not.toBeInTheDocument();
  });

  it("should open a valve with no state channel on its first click", async () => {
    const { sinks, factory } = createChannelTelem();
    const button = await renderValve(factory, { type: "channel", command: 7 });
    await vi.waitUntil(() => sinks.has(7));
    await page.elementLocator(button).click();
    await expect.poll(() => sinks.get(7)?.values).toEqual([1]);
    expect(page.getByRole("menuitem").elements()).toHaveLength(0);
  });

  it("should ask which command to send while its command channel disagrees", async () => {
    const { sinks, factory } = createChannelTelem({ 5: 0, 7: 1 });
    const button = await renderValve(factory, { type: "channel", key: 5, command: 7 });
    await expect
      .element(page.getByRole("img", { name: "Unknown" }))
      .not.toBeInTheDocument();
    await page.elementLocator(button).click();
    await page.getByRole("menuitem", { name: "Close" }).click();
    await expect.poll(() => sinks.get(7)?.values).toEqual([0]);
  });

  it("should ask which command to send for a value that matches neither", async () => {
    const onConfigChange = vi.fn();
    const Wrapper = createSynnaxWrapper({
      client: null,
      additionalRegistry: REGISTRY,
    });
    const { container } = await renderNode(
      Wrapper,
      {
        spec: { type: "builtin", name: "valve" },
        states: [
          {
            key: "active",
            source: { type: "local", value: 3 },
            match: { type: "exact" },
          },
        ],
      },
      onConfigChange,
    );
    const button = await vi.waitUntil(() => queryButton(container));
    await page.elementLocator(button).click();
    expect(onConfigChange).not.toHaveBeenCalled();
    await page.getByRole("menuitem", { name: "Open" }).click();
    expect(onConfigChange).toHaveBeenCalledExactlyOnceWith({
      states: [
        schematic.stateBindingZ.parse({
          key: "active",
          source: { type: "local", value: 1 },
          match: { type: "exact" },
        }),
      ],
    });
  });

  it("should ask which command to send for a click on a part", async () => {
    const client = createTestClient();
    const symbol = await client.schematics.symbols.create({
      name: "Lamp switch",
      data: {
        drawing: { type: "svg", markup: MARKUP },
        variant: "static",
        ...DATA,
        commands: [
          {
            key: "off",
            name: "Off",
            state: "active",
            value: { type: "fixed", value: 0 },
          },
          {
            key: "on",
            name: "On",
            state: "active",
            value: { type: "fixed", value: 1 },
          },
        ],
        interactions: [
          { type: "next", key: "switch", part: "lamp", commands: ["off", "on"] },
        ],
      },
    });
    const Wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: symbolNode.REGISTRY,
    });
    const onConfigChange = vi.fn();
    const { container } = await renderNode(
      Wrapper,
      {
        spec: { type: "custom", key: symbol.key },
        states: [{ key: "active", source: { type: "local", value: 3 } }],
      },
      onConfigChange,
    );
    const lamp = await vi.waitUntil(() => container.querySelector(".lamp"));
    await page.elementLocator(lamp).click();
    expect(onConfigChange).not.toHaveBeenCalled();
    await page.getByRole("menuitem", { name: "On" }).click();
    expect(onConfigChange).toHaveBeenCalledExactlyOnceWith({
      states: [
        schematic.stateBindingZ.parse({
          key: "active",
          source: { type: "local", value: 1 },
        }),
      ],
    });
  });

  it("should send the entry picked for a command that asks for its value", async () => {
    const client = createTestClient();
    const { key: enumeration } = await client.enumerations.create({
      name: "Mode",
      entries: [
        { key: "auto", value: 0, name: "Auto" },
        { key: "manual", value: 2, name: "Manual" },
      ],
    });
    const symbol = await createAsk(client, {
      type: "enumeration",
      key: "mode",
      name: "Mode",
      enumeration,
      match: { type: "exact" },
      default: 0,
    });
    const { onConfigChange } = await renderAsk(client, symbol);
    await page.getByRole("menuitem", { name: "Manual" }).click();
    expect(onConfigChange).toHaveBeenCalledExactlyOnceWith({
      states: [
        schematic.stateBindingZ.parse({
          key: "mode",
          source: { type: "local", value: 2 },
        }),
      ],
    });
  });

  it("should send the number typed for a command that asks for its value", async () => {
    const client = createTestClient();
    const symbol = await createAsk(client, {
      type: "number",
      key: "level",
      name: "Level",
      bounds: { lower: 0, upper: 100 },
    });
    const { onConfigChange, button } = await renderAsk(client, symbol);
    const set = page.getByRole("button", { name: "Set" });
    const field = page.getByRole("textbox");
    await expect.element(set).toHaveAttribute("aria-disabled", "true");
    await field.click();
    await userEvent.keyboard("{Enter}");
    expect(onConfigChange).not.toHaveBeenCalled();
    await field.fill("42");
    await set.click();
    expect(onConfigChange).toHaveBeenCalledExactlyOnceWith({
      states: [
        schematic.stateBindingZ.parse({
          key: "level",
          source: { type: "local", value: 42 },
        }),
      ],
    });
    await page.elementLocator(button).click();
    await expect.element(field).toHaveValue(null);
    await expect.element(set).toHaveAttribute("aria-disabled", "true");
  });

  it("should give an enumeration state with no enumeration a number field", async () => {
    const client = createTestClient();
    const symbol = await createAsk(client, {
      type: "enumeration",
      key: "mode",
      name: "Mode",
      match: { type: "exact" },
      default: 0,
    });
    const { onConfigChange } = await renderAsk(client, symbol);
    await page.getByRole("textbox").fill("3");
    await userEvent.keyboard("{Enter}");
    expect(onConfigChange).toHaveBeenCalledExactlyOnceWith({
      states: [
        schematic.stateBindingZ.parse({
          key: "mode",
          source: { type: "local", value: 3 },
        }),
      ],
    });
  });

  it("should show why the entries of its enumeration did not load", async () => {
    const client = createTestClient();
    const { key: enumeration } = await client.enumerations.create({
      name: "Mode",
      entries: [{ key: "auto", value: 0, name: "Auto" }],
    });
    const symbol = await createAsk(client, {
      type: "enumeration",
      key: "mode",
      name: "Mode",
      enumeration,
      match: { type: "exact" },
      default: 0,
    });
    // A user who can read the symbol but not its enumeration.
    const limited = await createTestClientWithPolicy(client, {
      name: "symbols",
      objects: [schematic.symbol.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    await renderAsk(limited, symbol);
    await expect
      .element(page.getByText("Failed to retrieve enumeration"))
      .toBeVisible();
    expect(page.getByRole("menuitem").elements()).toHaveLength(0);
  });

  // A shape body, and a letter in a span of a text.
  it.each(["t_junction", "flowmeter_orifice"])(
    "should fill the solid shapes of %s with the node color",
    async (name) => {
      const solid = await renderSolid(name, "#ff0000");
      await vi.waitFor(() =>
        expect(getComputedStyle(solid).fill).toBe("rgb(255, 0, 0)"),
      );
    },
  );

  it.each(["t_junction", "flowmeter_orifice"])(
    "should fill the solid shapes of %s with no node color",
    async (name) => {
      const solid = await renderSolid(name);
      expect(getComputedStyle(solid).fill).not.toBe("none");
    },
  );
});
