// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { assert, describe, expect, it } from "vitest";

import { lineplot } from "@/lineplot/aether";
import { Frame } from "@/lineplot/Frame";
import { rule } from "@/lineplot/rule/aether";
import { Rule } from "@/lineplot/rule/Rule";
import { Viewport } from "@/lineplot/Viewport";
import { render } from "@/testutil";

const RULE_PATH = [
  "alamos",
  "status",
  "synnax",
  "theming",
  "telem",
  "staleness",
  "render",
  "plot",
  "rule",
];

const POINTER = { pointerId: 1, button: 0, isPrimary: true };

// Moves are coalesced and delivered on the next animation frame.
const frame = async (): Promise<void> =>
  await new Promise((resolve) => requestAnimationFrame(() => resolve()));

const drag = async (el: Element, dy: number): Promise<void> => {
  fireEvent.pointerDown(el, { ...POINTER, clientX: 0, clientY: 0 });
  fireEvent.pointerMove(window, { ...POINTER, clientX: 0, clientY: dy });
  await act(frame);
  fireEvent.pointerUp(window, { ...POINTER, clientX: 0, clientY: dy });
};

const RemountableViewport = (): ReactElement => {
  const [key, setKey] = useState(0);
  return (
    <>
      <button onClick={() => setKey((k) => k + 1)}>remount</button>
      <Viewport key={key} />
    </>
  );
};

// The rule mounts before the viewport, as in a line plot.
const setup = async (viewport: ReactElement = <Viewport />) => {
  const rendered = render(
    <Frame aetherKey="plot">
      <Rule aetherKey="rule" position={1} color="#ff0000" />
      {viewport}
    </Frame>,
    { render: true, registry: lineplot.REGISTRY },
  );
  const leaf = await waitFor(() => {
    const found = rendered.root.findChildAtPath(RULE_PATH);
    assert(found instanceof rule.Rule);
    return found;
  });
  act(() => leaf.setState((p) => ({ ...p, pixelPosition: 10 })));
  await waitFor(() =>
    expect(rendered.container.querySelector(".pluto-rule")).not.toBeNull(),
  );
  return { ...rendered, leaf };
};

describe("Rule", () => {
  it("should render inside the plot viewport", async () => {
    const { container } = await setup();
    expect(
      container.querySelector(".pluto-line-plot__viewport .pluto-rule"),
    ).not.toBeNull();
  });

  it("should move into the viewport when the viewport mounts again", async () => {
    const { container, getByText } = await setup(<RemountableViewport />);
    const old = container.querySelector(".pluto-line-plot__viewport");
    fireEvent.click(getByText("remount"));
    await waitFor(() => {
      const viewport = container.querySelector(".pluto-line-plot__viewport");
      expect(viewport).not.toBe(old);
      expect(viewport?.querySelector(".pluto-rule")).not.toBeNull();
    });
  });

  it("should start each drag from where the previous drag ended", async () => {
    const { container, leaf } = await setup();
    const handle = container.querySelector(".pluto-rule__drag-handle");
    assert(handle != null);
    await drag(handle, 20);
    await waitFor(() => expect(leaf.state.pixelPosition).toBe(30));
    await drag(handle, 5);
    await waitFor(() => expect(leaf.state.pixelPosition).toBe(35));
  });
});
