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

const TEMP_KEY = "temp-uuid-1234";
const REAL_KEY = "real-cluster-key-5678";

const BASE_CLUSTER: Node.Cluster = {
  key: TEMP_KEY,
  name: "My Cluster",
  host: "example.com",
  port: 9090,
  username: "",
  password: "",
  secure: false,
};

describe("purgeDuplicateClusters", () => {
  it("should remove a temp-keyed cluster when a real key is set for the same host/port", () => {
    // Simulate Connect form saving a cluster with a temp key
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    expect(state.clusters[TEMP_KEY]).toBeDefined();

    // Simulate Login page setting the cluster with the real key
    state = Node.reducer(state, Node.set({ ...BASE_CLUSTER, key: REAL_KEY }));
    expect(state.clusters[REAL_KEY]).toBeDefined();
    expect(state.clusters[TEMP_KEY]).toBeUndefined();
  });

  it("should keep the newly set cluster and not the old duplicate", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(
      state,
      Node.set({
        ...BASE_CLUSTER,
        key: REAL_KEY,
        username: "synnax",
        password: "seldon",
      }),
    );
    expect(
      Object.keys(state.clusters).filter(
        (k) =>
          state.clusters[k].host === "example.com" && state.clusters[k].port === 9090,
      ),
    ).toHaveLength(1);
    expect(state.clusters[REAL_KEY].username).toBe("synnax");
  });

  it("should not purge clusters with different host/port", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(
      state,
      Node.set({ ...BASE_CLUSTER, key: REAL_KEY, host: "other.com" }),
    );
    expect(state.clusters[TEMP_KEY]).toBeDefined();
    expect(state.clusters[REAL_KEY]).toBeDefined();
  });

  it("should not purge a duplicate that differs only in secure flag", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(
      state,
      Node.set({ ...BASE_CLUSTER, key: REAL_KEY, secure: true }),
    );
    expect(state.clusters[TEMP_KEY]).toBeDefined();
    expect(state.clusters[REAL_KEY]).toBeDefined();
  });
});

describe("set", () => {
  it("should add a new cluster keyed by its key", () => {
    const state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    expect(state.clusters[TEMP_KEY]).toEqual(BASE_CLUSTER);
  });

  it("should overwrite an existing cluster with the same key", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(state, Node.set({ ...BASE_CLUSTER, name: "Renamed Inline" }));
    expect(state.clusters[TEMP_KEY].name).toBe("Renamed Inline");
  });
});

describe("remove", () => {
  it("should remove a single cluster by key", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(state, Node.remove(TEMP_KEY));
    expect(state.clusters[TEMP_KEY]).toBeUndefined();
  });

  it("should remove multiple clusters by key", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(
      state,
      Node.set({ ...BASE_CLUSTER, key: REAL_KEY, host: "other.com" }),
    );
    state = Node.reducer(state, Node.remove([TEMP_KEY, REAL_KEY]));
    expect(state.clusters[TEMP_KEY]).toBeUndefined();
    expect(state.clusters[REAL_KEY]).toBeUndefined();
  });

  it("should leave the default clusters untouched when removing a missing key", () => {
    const state = Node.reducer(Node.ZERO_SLICE_STATE, Node.remove("does-not-exist"));
    expect(Object.keys(state.clusters)).toEqual(
      Object.keys(Node.ZERO_SLICE_STATE.clusters),
    );
  });
});

describe("select / clearSelected", () => {
  it("should set the selected key", () => {
    const state = Node.reducer(Node.ZERO_SLICE_STATE, Node.select(TEMP_KEY));
    expect(state.selected).toBe(TEMP_KEY);
  });

  it("should clear the selected key", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.select(TEMP_KEY));
    state = Node.reducer(state, Node.clearSelected());
    expect(state.selected).toBeUndefined();
  });
});

describe("rename", () => {
  it("should rename a cluster in place", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(state, Node.rename({ key: TEMP_KEY, name: "New Name" }));
    expect(state.clusters[TEMP_KEY].name).toBe("New Name");
  });

  it("should throw when renaming to a name owned by another cluster", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(
      state,
      Node.set({ ...BASE_CLUSTER, key: REAL_KEY, host: "other.com" }),
    );
    expect(() =>
      Node.reducer(state, Node.rename({ key: REAL_KEY, name: BASE_CLUSTER.name })),
    ).toThrow(/already exists/);
  });
});

describe("changeKey", () => {
  it("should move a cluster from the old key to the new key", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(state, Node.changeKey({ oldKey: TEMP_KEY, newKey: REAL_KEY }));
    expect(state.clusters[TEMP_KEY]).toBeUndefined();
    expect(state.clusters[REAL_KEY]).toEqual({ ...BASE_CLUSTER, key: REAL_KEY });
  });

  it("should re-point the selected key when the changed cluster was selected", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(state, Node.select(TEMP_KEY));
    state = Node.reducer(state, Node.changeKey({ oldKey: TEMP_KEY, newKey: REAL_KEY }));
    expect(state.selected).toBe(REAL_KEY);
  });

  it("should leave the selected key untouched when a different cluster changes key", () => {
    let state = Node.reducer(Node.ZERO_SLICE_STATE, Node.set(BASE_CLUSTER));
    state = Node.reducer(state, Node.select("LOCAL"));
    state = Node.reducer(state, Node.changeKey({ oldKey: TEMP_KEY, newKey: REAL_KEY }));
    expect(state.selected).toBe("LOCAL");
  });
});
