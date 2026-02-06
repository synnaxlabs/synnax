// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Cluster, reducer, set, ZERO_SLICE_STATE } from "@/cluster/slice";

const TEMP_KEY = "temp-uuid-1234";
const REAL_KEY = "real-cluster-key-5678";

const BASE_CLUSTER: Cluster = {
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
    let state = reducer(ZERO_SLICE_STATE, set(BASE_CLUSTER));
    expect(state.clusters[TEMP_KEY]).toBeDefined();

    // Simulate Login page setting the cluster with the real key
    state = reducer(state, set({ ...BASE_CLUSTER, key: REAL_KEY }));
    expect(state.clusters[REAL_KEY]).toBeDefined();
    expect(state.clusters[TEMP_KEY]).toBeUndefined();
  });

  it("should keep the newly set cluster and not the old duplicate", () => {
    let state = reducer(ZERO_SLICE_STATE, set(BASE_CLUSTER));
    state = reducer(
      state,
      set({ ...BASE_CLUSTER, key: REAL_KEY, username: "synnax", password: "seldon" }),
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
    let state = reducer(ZERO_SLICE_STATE, set(BASE_CLUSTER));
    state = reducer(state, set({ ...BASE_CLUSTER, key: REAL_KEY, host: "other.com" }));
    expect(state.clusters[TEMP_KEY]).toBeDefined();
    expect(state.clusters[REAL_KEY]).toBeDefined();
  });
});
