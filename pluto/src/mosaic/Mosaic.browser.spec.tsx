// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/lyra/haul";
import { computed, renderThemed, resolve } from "@synnaxlabs/lyra/styletest";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Mosaic } from "@/mosaic";

const WIDTH = 800;
const HEIGHT = 500;

const Leaf = ({ nodeKey }: { nodeKey: number }): ReactElement => (
  <Mosaic.Leaf nodeKey={nodeKey} data-testid={`leaf-${nodeKey}`}>
    <Tabs.Frame initialValue="a" data-testid={`tabs-${nodeKey}`}>
      <Tabs.Selector>
        <Tabs.Tab itemKey="a">Tab a</Tabs.Tab>
      </Tabs.Selector>
      <Tabs.Content itemKey="a">Content {nodeKey}</Tabs.Content>
    </Tabs.Frame>
  </Mosaic.Leaf>
);

const renderMosaic = async (): Promise<Element> => {
  const { container } = await renderThemed(
    <div style={{ width: WIDTH, height: HEIGHT }}>
      <Haul.Provider>
        <Mosaic.Frame>
          <Mosaic.Split nodeKey={1}>
            <Leaf nodeKey={2} />
            <Leaf nodeKey={3} />
          </Mosaic.Split>
        </Mosaic.Frame>
      </Haul.Provider>
    </div>,
  );
  const frame = container.querySelector(`.${CSS.B("mosaic")}`);
  if (frame == null) throw new Error("mosaic frame not found");
  return frame;
};

const leaf = (key: number): Element => {
  const el = document.querySelector(`[data-testid="leaf-${key}"]`);
  if (el == null) throw new Error(`leaf ${key} not found`);
  return el;
};

describe("Mosaic", () => {
  it("should fill its container", async () => {
    const rect = (await renderMosaic()).getBoundingClientRect();
    expect(rect.width).toBe(WIDTH);
    expect(rect.height).toBe(HEIGHT);
  });

  it("should lay out as a block, not a flex box", async () => {
    expect(computed(await renderMosaic(), "display")).toBe("block");
  });

  it("should lift off the page with the lift shadow", async () => {
    expect(computed(await renderMosaic(), "box-shadow")).toBe(
      resolve("var(--pluto-shadow-lift)", "box-shadow"),
    );
  });

  it("should split its width between two full height leaves", async () => {
    const frame = (await renderMosaic()).getBoundingClientRect();
    const first = leaf(2).getBoundingClientRect();
    const second = leaf(3).getBoundingClientRect();
    expect(first.left).toBe(frame.left);
    expect(second.right).toBeCloseTo(frame.right, 0);
    expect(second.left).toBeGreaterThanOrEqual(first.right);
    for (const pane of [first, second]) {
      expect(pane.height).toBe(HEIGHT);
      expect(pane.width).toBeCloseTo(WIDTH / 2, -1);
    }
  });

  it("should fill each leaf with its tabs", async () => {
    await renderMosaic();
    const tabs = leaf(2).querySelector('[data-testid="tabs-2"]');
    if (tabs == null) throw new Error("tabs not found");
    const outer = leaf(2).getBoundingClientRect();
    const inner = tabs.getBoundingClientRect();
    expect(inner.width).toBe(outer.width);
    expect(inner.height).toBe(outer.height);
  });
});
