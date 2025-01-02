// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Node, sortAndSplice } from "@/tree/core";

describe("Tree", () => {
  describe("sortAndSplice", () => {
    it("should correctly temporarily force position of a node higher up than it should be", () => {
      const nodes: Node[] = [
        { key: "1", name: "1" },
        { key: "2", name: "2" },
        { key: "3", name: "3", forcePosition: 1 },
      ];
      const result = sortAndSplice(nodes, true);
      expect(result).toEqual([
        { key: "1", name: "1" },
        { key: "3", name: "3", forcePosition: 1 },
        { key: "2", name: "2" },
      ]);
    });
    it("should correctly temporarily force position of a node lower down than it should be", () => {
      const nodes: Node[] = [
        { key: "1", name: "1", forcePosition: 1 },
        { key: "2", name: "2" },
        { key: "3", name: "3" },
      ];
      const result = sortAndSplice(nodes);
      expect(result).toEqual([
        { key: "2", name: "2" },
        { key: "1", name: "1", forcePosition: 1 },
        { key: "3", name: "3" },
      ]);
    });
  });
});
