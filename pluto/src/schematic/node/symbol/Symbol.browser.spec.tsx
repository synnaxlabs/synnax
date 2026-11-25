// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { ReactFlowProvider } from "@xyflow/react";
import { assert, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-react";
import type { z } from "zod";

import { Node } from "@/schematic/node";
import { createAsyncSynnaxWrapper } from "@/testutil";
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

type Config = z.input<typeof schematic.symbolNodeConfigZ>;

const renderNode = async (client: Synnax, config: Omit<Config, "variant">) => {
  const Wrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: symbolNode.REGISTRY,
  });
  const { container } = await render(
    <Wrapper>
      <Node.Custom.MotionProvider>
        <ReactFlowProvider>
          <Node.REGISTRY.symbol.Node
            nodeKey="s1"
            selected={false}
            onConfigChange={vi.fn()}
            config={schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config })}
          />
        </ReactFlowProvider>
      </Node.Custom.MotionProvider>
    </Wrapper>,
  );
  return container;
};

const renderLamp = async (config: Pick<Config, "motionDisabled"> = {}) => {
  const client = createTestClient();
  const symbol = await client.schematics.symbols.create({
    name: "Lamp",
    data: { drawing: { type: "svg", markup: MARKUP }, variant: "static", ...DATA },
  });
  const container = await renderNode(client, {
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

  it("should draw a copy of a built-in as wide as the built-in", async () => {
    const client = createTestClient();
    const data = Node.BUILT_INS.get("ball_valve");
    assert(data != null);
    const copy = await client.schematics.symbols.create({ name: "Ball (copy)", data });
    const widthOf = async (spec: Config["spec"]) => {
      const container = await renderNode(client, { spec });
      const circle = await vi.waitUntil(() => container.querySelector("circle"));
      return getComputedStyle(circle).strokeWidth;
    };
    expect([
      await widthOf({ type: "custom", key: copy.key }),
      await widthOf({ type: "builtin", name: "ball_valve" }),
    ]).toEqual(["2px", "2px"]);
  });
});
