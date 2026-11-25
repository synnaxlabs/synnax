// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { createTestClient, SyntheticCore } from "@synnaxlabs/client/testutil";
import { Rate, TimeStamp } from "@synnaxlabs/x";
import { ReactFlowProvider } from "@xyflow/react";
import type { FC, PropsWithChildren } from "react";
import { describe, expect, it, vi } from "vitest";
import { page } from "vitest/browser";
import { render } from "vitest-browser-react";
import type { z } from "zod";

import { Node } from "@/schematic/node";
import { createCommandSinks } from "@/schematic/node/testutil";
import { telemTest } from "@/telem/aether/testutil";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";

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
    additionalRegistry: symbolNode.REGISTRY,
  });
  const { container } = await renderNode(Wrapper, {
    spec: { type: "custom", key: symbol.key },
    states: [{ key: "active", source: { type: "local", value: 1 } }],
    ...config,
  });
  return await vi.waitUntil(() => container.querySelector(".lamp"));
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

  it("should ask which command to send while its state is unknown", async () => {
    const now = TimeStamp.now();
    const core = new SyntheticCore({ rate: Rate.hz(10), start: now, now });
    const { sinks, factory } = createCommandSinks();
    const Wrapper = createSynnaxWrapper({
      client: null,
      additionalRegistry: symbolNode.REGISTRY,
      telemFactories: [telemTest.createRemoteFactory(core), factory],
    });
    const { container } = await renderNode(Wrapper, {
      spec: { type: "builtin", name: "valve" },
      states: [
        {
          key: "active",
          source: { type: "channel", key: core.data[0].key, command: 7 },
        },
      ],
    });
    await expect.element(page.getByRole("img", { name: "Unknown" })).toBeVisible();
    await vi.waitUntil(() => sinks.has(7));
    const button = await vi.waitUntil(() => container.querySelector("button"));
    await page.elementLocator(button).click();
    const open = page.getByRole("menuitem", { name: "Open" });
    await expect.element(page.getByRole("menuitem", { name: "Close" })).toBeVisible();
    await open.click();
    await expect.poll(() => sinks.get(7)?.values).toEqual([1]);
    await expect.element(open).not.toBeInTheDocument();
  });

  it("should ask which command to send for a value that matches neither", async () => {
    const onConfigChange = vi.fn();
    const Wrapper = createSynnaxWrapper({
      client: null,
      additionalRegistry: symbolNode.REGISTRY,
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
    const button = await vi.waitUntil(() => container.querySelector("button"));
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
});
