// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-react";
import type { z } from "zod";

import { Node } from "@/schematic/node";
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

const renderLamp = async (
  config: Pick<z.input<typeof schematic.symbolNodeConfigZ>, "motionDisabled"> = {},
) => {
  const client = createTestClient();
  const symbol = await client.schematics.symbols.create({
    name: "Lamp",
    data: { drawing: { type: "svg", markup: MARKUP }, variant: "static", ...DATA },
  });
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
            config={schematic.symbolNodeConfigZ.parse({
              variant: "symbol",
              spec: { type: "custom", key: symbol.key },
              states: [{ key: "active", source: { type: "local", value: 1 } }],
              ...config,
            })}
          />
        </ReactFlowProvider>
      </Node.Custom.MotionProvider>
    </Wrapper>,
  );
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

  // A shape body and a text, which the theme fills unless it sets its own fill.
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
