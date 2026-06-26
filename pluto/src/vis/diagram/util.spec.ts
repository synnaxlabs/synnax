// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type InternalNode, type NodeChange as RFNodeChange } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import {
  internalNodeBox,
  partitionNodeChanges,
  resolveEndpoint,
} from "@/vis/diagram/util";

const node = (
  positionAbsolute: xy.XY,
  measured: { width?: number; height?: number },
): InternalNode =>
  ({ measured, internals: { positionAbsolute } }) as unknown as InternalNode;

const handle = (id: string, position: string) => ({
  id,
  position,
  x: 40,
  y: 10,
  width: 8,
  height: 8,
});

const internals = (
  positionAbsolute: xy.XY,
  ...source: ReturnType<typeof handle>[]
) => ({
  positionAbsolute,
  handleBounds: { source, target: [] },
});

describe("internalNodeBox", () => {
  it("builds a flow-coordinate box from the node's absolute position and dimensions", () => {
    const b = internalNodeBox(node({ x: 500, y: 80 }, { width: 40, height: 30 }));
    expect(box.topLeft(b)).toEqual(xy.construct(500, 80));
    expect(box.width(b)).toEqual(40);
    expect(box.height(b)).toEqual(30);
  });

  it("returns box.ZERO when the node is null", () => {
    expect(internalNodeBox(null)).toEqual(box.ZERO);
  });

  it("returns box.ZERO when the node has not been measured yet", () => {
    expect(internalNodeBox(node({ x: 500, y: 80 }, {}))).toEqual(box.ZERO);
  });
});

describe("resolveEndpoint", () => {
  it.each([
    { side: "right", expected: { x: 148, y: 64 } },
    { side: "top", expected: { x: 144, y: 60 } },
    { side: "bottom", expected: { x: 144, y: 68 } },
    { side: "left", expected: { x: 140, y: 64 } },
  ])("places the point on the $side edge of the handle", ({ side, expected }) => {
    const geom = internals({ x: 100, y: 50 }, handle("a", side));
    expect(resolveEndpoint(geom, "a")).toEqual({
      position: expected,
      orientation: side,
    });
  });

  it("returns null when the named handle is absent", () => {
    const geom = internals({ x: 100, y: 50 }, handle("a", "right"));
    expect(resolveEndpoint(geom, "missing")).toBeNull();
  });

  it("uses the first handle when no handle is named", () => {
    const geom = internals(
      { x: 100, y: 50 },
      handle("a", "left"),
      handle("b", "right"),
    );
    expect(resolveEndpoint(geom, "")?.orientation).toEqual("left");
  });
});

const dimensionsChange = (
  id: string,
  dimensions: { width: number; height: number } | null,
  resizing = false,
): RFNodeChange =>
  ({ type: "dimensions", id, dimensions, resizing }) as unknown as RFNodeChange;

describe("partitionNodeChanges", () => {
  it("collects finalized dimension changes into sizes", () => {
    const { sizes, passthrough, removed } = partitionNodeChanges([
      dimensionsChange("a", { width: 40, height: 30 }),
    ]);
    expect(sizes).toEqual([["a", { width: 40, height: 30 }]]);
    expect(passthrough).toHaveLength(0);
    expect(removed).toHaveLength(0);
  });

  it("ignores dimension changes that are still resizing", () => {
    const { sizes } = partitionNodeChanges([
      dimensionsChange("a", { width: 40, height: 30 }, true),
    ]);
    expect(sizes).toHaveLength(0);
  });

  it("ignores dimension changes with a zero width or height", () => {
    const { sizes } = partitionNodeChanges([
      dimensionsChange("a", { width: 0, height: 30 }),
      dimensionsChange("b", { width: 40, height: 0 }),
    ]);
    expect(sizes).toHaveLength(0);
  });

  it("ignores dimension changes with null dimensions", () => {
    const { sizes } = partitionNodeChanges([dimensionsChange("a", null)]);
    expect(sizes).toHaveLength(0);
  });

  it("collects removals and passes them through", () => {
    const change = { type: "remove", id: "a" } as unknown as RFNodeChange;
    const { removed, passthrough } = partitionNodeChanges([change]);
    expect(removed).toEqual(["a"]);
    expect(passthrough).toEqual([change]);
  });

  it("passes non-dimension changes through", () => {
    const change = { type: "position", id: "a" } as unknown as RFNodeChange;
    const { passthrough, sizes, removed } = partitionNodeChanges([change]);
    expect(passthrough).toEqual([change]);
    expect(sizes).toHaveLength(0);
    expect(removed).toHaveLength(0);
  });
});
