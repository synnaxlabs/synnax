// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Node } from "@/session/node";

const CLUSTER_A: Node.Node = {
  key: "a",
  name: "Alpha",
  host: "a.example.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

const CLUSTER_B: Node.Node = {
  key: "b",
  name: "Beta",
  host: "b.example.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: true,
};

const storeWith = (clusters: Node.Node[], selected?: string): Node.StoreState => ({
  [Node.SLICE_NAME]: {
    version: 0,
    selected,
    nodes: Object.fromEntries(clusters.map((c) => [c.key, c])),
  },
});

describe("cluster selectors", () => {
  describe("selectSelectedKey", () => {
    it("should return the selected key", () => {
      expect(Node.selectSelectedKey(storeWith([CLUSTER_A], "a"))).toBe("a");
    });

    it("should return undefined when nothing is selected", () => {
      expect(Node.selectSelectedKey(storeWith([CLUSTER_A]))).toBeUndefined();
    });
  });

  describe("selectState", () => {
    it("should resolve a cluster by explicit key", () => {
      expect(Node.selectState(storeWith([CLUSTER_A, CLUSTER_B]), "b")).toEqual(
        CLUSTER_B,
      );
    });

    it("should fall back to the selected cluster when no key is given", () => {
      expect(Node.selectState(storeWith([CLUSTER_A, CLUSTER_B], "a"))).toEqual(
        CLUSTER_A,
      );
    });

    it("should return undefined when neither a key nor a selection resolves", () => {
      expect(Node.selectState(storeWith([CLUSTER_A]))).toBeUndefined();
    });
  });

  describe("selectMany", () => {
    it("should return every cluster when no keys are given", () => {
      expect(Node.selectMany(storeWith([CLUSTER_A, CLUSTER_B]))).toEqual([
        CLUSTER_A,
        CLUSTER_B,
      ]);
    });

    it("should filter to the requested keys", () => {
      expect(Node.selectMany(storeWith([CLUSTER_A, CLUSTER_B]), ["b"])).toEqual([
        CLUSTER_B,
      ]);
    });
  });

  describe("selectAllNames", () => {
    it("should return the names of every cluster", () => {
      expect(Node.selectAllNames(storeWith([CLUSTER_A, CLUSTER_B]))).toEqual([
        "Alpha",
        "Beta",
      ]);
    });
  });

  describe("selectIsAnySelected", () => {
    it("should be true when a cluster is selected", () => {
      expect(Node.selectIsAnySelected(storeWith([CLUSTER_A], "a"))).toBe(true);
    });

    it("should be false when nothing is selected", () => {
      expect(Node.selectIsAnySelected(storeWith([CLUSTER_A]))).toBe(false);
    });
  });
});
