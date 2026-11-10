// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, spyOnSend } from "@synnaxlabs/client/testutil";
import { mockBoundingClientRect } from "@synnaxlabs/lyra/testutil";
import { uuid, xy } from "@synnaxlabs/x";
import { act, fireEvent, render, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { diagram } from "@/vis/diagram/aether";
import { canvasTest } from "@/vis/render/testutil";

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

const renderSchematic = async (): Promise<{
  key: string;
  utils: ReturnType<typeof render>;
}> => {
  const project = await client.projects.create({ name: `project_${uuid.create()}` });
  const { key } = await client.schematics.create({
    project: project.key,
    name: "schematic",
    nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
    configs: { n1: { variant: "tank", label: { label: "Pump" } } },
  });
  const wrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: diagram.REGISTRY,
    renderContext: canvasTest.record(),
    telemFactories: [],
  });
  const View = (): ReactElement => (
    <Errors.SuspenseBoundary loading={null}>
      <Schematic.Suspended schematicKey={key}>
        <Schematic.Schematic
          editable
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
            set: { path: ["configs", "n1", "label", "label"], value: "Valve" },
          },
        ],
      }),
    );
    await waitFor(async () =>
      expect((await client.schematics.retrieve({ key })).configs.n1).toMatchObject({
        label: { label: "Valve" },
      }),
    );
  });
});
