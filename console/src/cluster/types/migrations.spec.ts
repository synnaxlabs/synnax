// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { migrateSlice, ZERO_SLICE_STATE } from "@/cluster/types";
import * as v0 from "@/cluster/types/v0";
import * as v1 from "@/cluster/types/v1";
import * as v2 from "@/cluster/types/v2";
import * as v3 from "@/cluster/types/v3";

const STATES = [
  v0.ZERO_SLICE_STATE,
  v1.ZERO_SLICE_STATE,
  v2.ZERO_SLICE_STATE,
  v3.ZERO_SLICE_STATE,
];

describe("migrations", () => {
  STATES.forEach((state) =>
    it(`should migrate slice from ${state.version} to latest`, () =>
      expect(migrateSlice(state)).toEqual(ZERO_SLICE_STATE)),
  );

  it("should assign a uuid to clusters with empty keys", () => {
    const stateWithEmptyKey: v2.SliceState = {
      ...v2.ZERO_SLICE_STATE,
      clusters: {
        ...v2.ZERO_SLICE_STATE.clusters,
        "": {
          key: "",
          name: "Bad Cluster",
          host: "example.com",
          port: 9090,
          username: "",
          password: "",
          secure: false,
        },
      },
    };
    const migrated = migrateSlice(stateWithEmptyKey);
    expect(migrated.clusters[""]).toBeUndefined();
    const clusterKeys = Object.keys(migrated.clusters);
    const newKey = clusterKeys.find(
      (k) => !Object.keys(v2.ZERO_SLICE_STATE.clusters).includes(k),
    );
    expect(newKey).toBeDefined();
    expect(newKey!.length).toBeGreaterThan(0);
    expect(migrated.clusters[newKey!].name).toBe("Bad Cluster");
    expect(migrated.clusters[newKey!].key).toBe(newKey);
  });
});
