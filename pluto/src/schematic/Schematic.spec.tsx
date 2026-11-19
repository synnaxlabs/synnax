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
import { type ReactElement } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { control } from "@/telem/control/aether";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { diagram } from "@/vis/diagram/aether";
import { latestSample } from "@/vis/latestSample/aether";
import { canvasTest } from "@/vis/render/testutil";
import { Staleness } from "@/vis/staleness";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { toggle } from "@/vis/toggle/aether";

const client = createTestClient();

const RECT = mockBoundingClientRect(0, 0, 1000, 500)();

// The @juggle polyfill reads computed styles, which jsdom leaves empty, so it never
// reports a size. Nodes stay unmeasured: React Flow reads their transforms through
// DOMMatrixReadOnly, which jsdom lacks.
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
  badged?: boolean;
  registry?: aether.ComponentRegistry;
}

const renderSchematic = async (
  nodes: schematic.New["nodes"] = PUMP,
  { editable = true, badged = false, registry = {} }: RenderOptions = {},
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
        <Staleness.Provider badged={badged}>
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
        </Staleness.Provider>
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

class StubChip extends aether.Leaf<typeof looseZ> {
  static readonly TYPE = control.Chip.TYPE;
  schema = looseZ;
}

class StubIndicator extends aether.Leaf<typeof looseZ> {
  static readonly TYPE = control.Indicator.TYPE;
  schema = looseZ;
}

class StubLatestSample extends aether.Leaf<typeof looseZ> {
  static readonly TYPE = latestSample.LatestSample.TYPE;
  schema = looseZ;
}

const STALE_REGISTRY: aether.ComponentRegistry = {
  ...stalenessTest.createRegistry(toggle.Toggle.TYPE, toggle.toggleStateZ),
  [StubChip.TYPE]: StubChip,
  [StubIndicator.TYPE]: StubIndicator,
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
      { editable: false, badged: true, registry: STALE_REGISTRY },
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
});
