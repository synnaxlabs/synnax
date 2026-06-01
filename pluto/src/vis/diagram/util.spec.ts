// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type InternalNode } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { internalNodeBox } from "@/vis/diagram/util";

const node = (
  positionAbsolute: xy.XY,
  measured: { width?: number; height?: number },
): InternalNode =>
  ({ measured, internals: { positionAbsolute } }) as unknown as InternalNode;

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
