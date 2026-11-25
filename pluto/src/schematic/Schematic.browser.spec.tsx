// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type schematic } from "@synnaxlabs/client";
import { createTestClient, spyOnSend } from "@synnaxlabs/client/testutil";
import { mockBoundingClientRect } from "@synnaxlabs/lyra/testutil";
import { id, uuid, xy } from "@synnaxlabs/x";
import { act, fireEvent, render, waitFor, within } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { controlTest } from "@/telem/control/aether/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil";
import { diagram } from "@/vis/diagram/aether";
import { latestSample } from "@/vis/latestSample/aether";
import { canvasTest } from "@/vis/render/testutil";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";
import { toggle } from "@/vis/toggle/aether";

const client = createTestClient();

const RECT = mockBoundingClientRect(0, 0, 1000, 500)();

// Notifies at once, so the schematic measures the fixed box below. Nodes stay
// unmeasured.
class ImmediateResizeObserver {
  constructor(private readonly notify: (entries: ResizeObserverEntry[]) => void) {}
  observe(el: Element): void {
    if (!el.classList.contains("react-flow__node")) this.notify([]);
  }
  unobserve(): void {}
  disconnect(): void {}
}

const PUMP: schematic.New["nodes"] = [
  {
    key: "n1",
    position: { x: 0, y: 0 },
    config: { variant: "tank", label: { label: "Pump" } },
  },
];

interface RenderOptions {
  editable?: boolean;
  registry?: aether.ComponentRegistry;
}

const renderSchematic = async (
  nodes: schematic.New["nodes"] = PUMP,
  { editable = true, registry = {} }: RenderOptions = {},
): Promise<{
  key: string;
  utils: ReturnType<typeof render>;
}> => {
  const project = await client.projects.create({ name: `project_${uuid.create()}` });
  const { key } = await client.schematics.create({
    project: project.key,
    name: "schematic",
    nodes,
  });
  const wrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: { ...diagram.REGISTRY, ...registry },
    renderContext: canvasTest.record(),
    telemFactories: [],
  });
  const View = (): ReactElement => (
    <Errors.SuspenseBoundary loading={null}>
      <Schematic.Suspended schematicKey={key}>
        <Schematic.Schematic
          editable={editable}
          visible
          viewport={{ position: xy.ZERO, zoom: 1 }}
          onViewportChange={() => {}}
          onEditableChange={() => {}}
          fitViewOnResize={false}
          setFitViewOnResize={() => {}}
          viewportMode="zoom"
          onViewportModeChange={() => {}}
          selected={[]}
          onSelectionChange={() => {}}
        />
      </Schematic.Suspended>
    </Errors.SuspenseBoundary>
  );
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(<View />, { wrapper });
  });
  return { key, utils };
};

// Stand in for the worker components of a toggle tooltip.
const looseZ = z.object({}).loose();

class StubLatestSample extends aether.Leaf<typeof looseZ> {
  static readonly TYPE = latestSample.LatestSample.TYPE;
  override schema = looseZ;
}

const STALE_REGISTRY: aether.ComponentRegistry = {
  ...stalenessTest.createRegistry(toggle.Toggle.TYPE, toggle.toggleStateZ),
  ...controlTest.createRegistry(),
  [StubLatestSample.TYPE]: StubLatestSample,
};

describe("Schematic", () => {
  // Canvas.useRegion measures against the lower2d canvas and bails without it.
  const canvas = document.createElement("div");

  beforeAll(() => {
    vi.stubGlobal("ResizeObserver", ImmediateResizeObserver);
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(RECT);
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(RECT.width);
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(RECT.height);
    canvas.className = "pluto-canvas--lower2d";
    document.body.appendChild(canvas);
  });

  afterAll(() => {
    canvas.remove();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("should send a label edit on the canvas as a path action", async () => {
    const { key, utils } = await renderSchematic();
    const label = await utils.findByText("Pump");
    const send = spyOnSend(client);
    // React Flow hides the nodes the stub leaves unmeasured, and a hidden element
    // reads no innerText.
    label
      .closest<HTMLElement>(".react-flow__node")
      ?.style.setProperty("visibility", "visible");
    fireEvent.dblClick(label);
    label.innerText = "Valve";
    fireEvent.keyDown(label, { key: "Enter" });
    await waitFor(() =>
      expect(
        send.mock.calls.find(([target]) => target === "/schematic/dispatch")?.[1],
      ).toMatchObject({
        actions: [
          {
            type: "set",
            set: {
              path: ["nodes", "n1", "config", "label", "label"],
              value: "Valve",
            },
          },
        ],
      }),
    );
    await waitFor(async () =>
      expect((await client.schematics.retrieve({ key })).nodes[0].config).toMatchObject(
        { label: { label: "Valve" } },
      ),
    );
  });

  it("should size a group box from the nodes it holds", async () => {
    const { utils } = await renderSchematic([
      { key: "g1", position: { x: -20, y: -40 }, config: { variant: "group_box" } },
      {
        key: "n1",
        position: { x: 0, y: 0 },
        config: { variant: "tank" },
        parent: "g1",
      },
      {
        key: "n2",
        position: { x: 100, y: 60 },
        config: { variant: "tank" },
        parent: "g1",
      },
    ]);
    // Nodes stay unmeasured, so the far corner is the position of n2.
    await waitFor(() => {
      const box = utils.container.querySelector<HTMLElement>(".pluto-group-box");
      expect(box?.style.width).toBe("140px");
      expect(box?.style.height).toBe("120px");
    });
  });

  it("should draw the label of a symbol node", async () => {
    const { utils } = await renderSchematic(
      [
        {
          key: "n1",
          position: { x: 0, y: 0 },
          config: {
            variant: "symbol",
            label: { label: "Heater" },
            spec: { type: "builtin", name: "valve" },
          },
        },
      ],
      { registry: symbolNode.REGISTRY },
    );
    expect(await utils.findByText("Heater")).toBeDefined();
  });

  it("should keep the tooltip over the stale badge and close it over the chip", async () => {
    const command = await client.channels.create({
      name: id.create(),
      dataType: DataType.UINT8,
      virtual: true,
    });
    const { utils } = await renderSchematic(
      [
        {
          key: "v1",
          position: xy.ZERO,
          config: { variant: "valve", commandChannel: command.key },
        },
      ],
      { editable: false, registry: STALE_REGISTRY },
    );
    // React Flow hides unmeasured nodes, so role queries cannot see the badge.
    const badge = await waitFor(() => {
      const el = utils.container.querySelector<HTMLElement>(".pluto-stale-badge");
      if (el == null) throw new Error("no stale badge yet");
      return el;
    });
    fireEvent.mouseMove(badge);
    await utils.findByText(command.name, undefined, { timeout: 3000 });
    const slot = badge.closest<HTMLElement>(".pluto-control-state");
    if (slot == null) throw new Error("the badge is not in the control slot");
    fireEvent.mouseMove(within(slot).getAllByRole("button", { hidden: true })[0]);
    await waitFor(() => expect(utils.queryByText(command.name)).toBeNull());
  });

  it.each([
    ["off", false, "all"],
    ["on", true, "none"],
  ])(
    "should give a part with a click pointer events %s edit mode",
    async (_, editable, pointerEvents) => {
      const symbol = await client.schematics.symbols.create({
        name: "Lamp",
        data: {
          drawing: {
            type: "svg",
            markup:
              '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
              '<rect class="lamp" width="10" height="10" /></svg>',
          },
          variant: "static",
          parts: [{ key: "lamp", name: "Lamp", selectors: [".lamp"] }],
          states: [{ type: "number", key: "level", name: "Level" }],
          commands: [
            {
              key: "on",
              name: "On",
              state: "level",
              value: { type: "fixed", value: 1 },
            },
          ],
          interactions: [{ type: "fire", key: "lamp", part: "lamp", command: "on" }],
        },
      });
      const { utils } = await renderSchematic(
        [
          {
            key: "l1",
            position: xy.ZERO,
            config: {
              variant: "symbol",
              spec: { type: "custom", key: symbol.key },
              states: [{ key: "level", source: { type: "local", value: 0 } }],
            },
          },
        ],
        { editable, registry: symbolNode.REGISTRY },
      );
      const lamp = await waitFor(() => {
        const el = utils.container.querySelector(".pluto-symbol__part--clickable");
        if (el == null) throw new Error("no clickable part yet");
        return el;
      });
      expect(getComputedStyle(lamp).pointerEvents).toBe(pointerEvents);
    },
  );

  it("should hide the tooltip while a node shows its command menu", async () => {
    const { utils } = await renderSchematic(
      [
        {
          key: "v1",
          position: xy.ZERO,
          config: {
            variant: "symbol",
            spec: { type: "builtin", name: "valve" },
            // A value that matches neither command makes a click open the menu.
            states: [
              {
                key: "active",
                source: { type: "local", value: 3 },
                match: { type: "exact" },
              },
            ],
          },
        },
      ],
      { editable: false, registry: symbolNode.REGISTRY },
    );
    // React Flow hides unmeasured nodes, so role queries cannot see the button.
    const button = await waitFor(() => {
      const el = utils.container.querySelector<HTMLElement>(".react-flow__node button");
      if (el == null) throw new Error("no symbol button yet");
      return el;
    });
    fireEvent.mouseMove(button);
    await utils.findByText("Stale timeout", undefined, { timeout: 3000 });
    fireEvent.click(button);
    const close = await utils.findByRole("menuitem", { name: "Close", hidden: true });
    expect(utils.queryByText("Stale timeout")).toBeNull();
    fireEvent.click(close);
    await utils.findByText("Stale timeout", undefined, { timeout: 3000 });
  });
});
