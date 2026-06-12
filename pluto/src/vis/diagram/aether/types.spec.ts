// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";
import { type NodeChange as RFNodeChange } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { translateNodeChangeForward } from "@/vis/diagram/aether/types";

describe("translateNodeChangeForward", () => {
  describe("dimensions", () => {
    it("forwards resizing when React Flow flags an active drag", () => {
      const change: RFNodeChange = {
        id: "n1",
        type: "dimensions",
        dimensions: { width: 40, height: 30 },
        resizing: true,
      };
      expect(translateNodeChangeForward(change)).toEqual({
        type: "dimensions",
        key: "n1",
        dimensions: { width: 40, height: 30 },
        resizing: true,
      });
    });

    it("defaults resizing to false when React Flow omits it", () => {
      const change: RFNodeChange = {
        id: "n1",
        type: "dimensions",
        dimensions: { width: 40, height: 30 },
      };
      expect(translateNodeChangeForward(change)).toEqual({
        type: "dimensions",
        key: "n1",
        dimensions: { width: 40, height: 30 },
        resizing: false,
      });
    });

    it("returns null when no dimensions are present", () => {
      const change: RFNodeChange = { id: "n1", type: "dimensions" };
      expect(translateNodeChangeForward(change)).toBeNull();
    });
  });

  describe("position", () => {
    it("forwards the position and dragging flag", () => {
      const change: RFNodeChange = {
        id: "n1",
        type: "position",
        position: { x: 10, y: 20 },
        dragging: true,
      };
      expect(translateNodeChangeForward(change)).toEqual({
        type: "position",
        key: "n1",
        position: xy.construct(10, 20),
        dragging: true,
      });
    });

    it("defaults dragging to false and returns null without a position", () => {
      const change: RFNodeChange = { id: "n1", type: "position" };
      expect(translateNodeChangeForward(change)).toBeNull();
    });
  });

  it("forwards remove changes", () => {
    const change: RFNodeChange = { id: "n1", type: "remove" };
    expect(translateNodeChangeForward(change)).toEqual({
      type: "remove",
      key: "n1",
    });
  });

  it("forwards select changes", () => {
    const change: RFNodeChange = { id: "n1", type: "select", selected: true };
    expect(translateNodeChangeForward(change)).toEqual({
      type: "select",
      key: "n1",
      selected: true,
    });
  });
});
