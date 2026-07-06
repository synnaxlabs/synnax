// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Cluster } from "@/session/cluster";

const CLUSTER_A: Cluster.Cluster = {
  key: "a",
  name: "Alpha",
  host: "a.example.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

const CLUSTER_B: Cluster.Cluster = {
  key: "b",
  name: "Beta",
  host: "b.example.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: true,
};

const storeWith = (
  clusters: Cluster.Cluster[],
  selected?: string,
): Cluster.StoreState => ({
  [Cluster.SLICE_NAME]: {
    version: 0,
    selected,
    clusters: Object.fromEntries(clusters.map((c) => [c.key, c])),
  },
});

describe("cluster selectors", () => {
  describe("selectSelectedKey", () => {
    it("should return the selected key", () => {
      expect(Cluster.selectSelectedKey(storeWith([CLUSTER_A], "a"))).toBe("a");
    });

    it("should return undefined when nothing is selected", () => {
      expect(Cluster.selectSelectedKey(storeWith([CLUSTER_A]))).toBeUndefined();
    });
  });

  describe("selectState", () => {
    it("should resolve a cluster by explicit key", () => {
      expect(Cluster.selectState(storeWith([CLUSTER_A, CLUSTER_B]), "b")).toEqual(
        CLUSTER_B,
      );
    });

    it("should fall back to the selected cluster when no key is given", () => {
      expect(Cluster.selectState(storeWith([CLUSTER_A, CLUSTER_B], "a"))).toEqual(
        CLUSTER_A,
      );
    });

    it("should return undefined when neither a key nor a selection resolves", () => {
      expect(Cluster.selectState(storeWith([CLUSTER_A]))).toBeUndefined();
    });
  });

  describe("selectMany", () => {
    it("should return every cluster when no keys are given", () => {
      expect(Cluster.selectMany(storeWith([CLUSTER_A, CLUSTER_B]))).toEqual([
        CLUSTER_A,
        CLUSTER_B,
      ]);
    });

    it("should filter to the requested keys", () => {
      expect(Cluster.selectMany(storeWith([CLUSTER_A, CLUSTER_B]), ["b"])).toEqual([
        CLUSTER_B,
      ]);
    });
  });

  describe("selectAllNames", () => {
    it("should return the names of every cluster", () => {
      expect(Cluster.selectAllNames(storeWith([CLUSTER_A, CLUSTER_B]))).toEqual([
        "Alpha",
        "Beta",
      ]);
    });
  });

  describe("selectIsAnySelected", () => {
    it("should be true when a cluster is selected", () => {
      expect(Cluster.selectIsAnySelected(storeWith([CLUSTER_A], "a"))).toBe(true);
    });

    it("should be false when nothing is selected", () => {
      expect(Cluster.selectIsAnySelected(storeWith([CLUSTER_A]))).toBe(false);
    });
  });
});
