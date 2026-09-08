// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Core } from "@/session/core";

const BASE: Omit<Core.Core, "key"> = {
  name: "My Core",
  host: "example.com",
  port: 9090,
  username: "",
  password: "",
  secure: false,
};

const ALPHA: Core.Core = { ...BASE, key: "alpha", name: "Alpha" };
const BRAVO: Core.Core = { ...BASE, key: "bravo", name: "Bravo", port: 9091 };

const reduce = (state: Core.SliceState, action: Core.Action): Core.SliceState =>
  Core.reducer(state, action);

const withCores = (...cores: Core.Core[]): Core.SliceState => ({
  ...Core.ZERO_SLICE_STATE,
  cores: Object.fromEntries(cores.map((c) => [c.key, c])),
});

describe("set", () => {
  it("should generate a key for a Core the user has just added", () => {
    const state = reduce(Core.ZERO_SLICE_STATE, Core.set(BASE));
    const added = Object.values(state.cores).find((c) => c.name === BASE.name);
    expect(added?.key).toBeTruthy();
  });

  it("should keep two Cores at one address apart", () => {
    let state = reduce(Core.ZERO_SLICE_STATE, Core.set({ ...BASE, name: "A" }));
    state = reduce(state, Core.set({ ...BASE, name: "B" }));
    const atAddress = Object.values(state.cores).filter(
      ({ host, port }) => host === BASE.host && port === BASE.port,
    );
    expect(atAddress).toHaveLength(2);
  });

  it("should replace the Core stored under an explicit key", () => {
    const state = reduce(
      withCores(ALPHA),
      Core.set({ ...BASE, key: ALPHA.key, name: "Renamed" }),
    );
    expect(Object.keys(state.cores)).toEqual([ALPHA.key]);
    expect(state.cores[ALPHA.key].name).toBe("Renamed");
  });

  it("should keep the key when an edit changes the address", () => {
    const state = reduce(
      withCores(ALPHA),
      Core.set({ ...ALPHA, port: 9099, key: ALPHA.key }),
    );
    expect(state.cores[ALPHA.key].port).toBe(9099);
  });

  it("should keep the cached cluster when an edit leaves the address alone", () => {
    const state = reduce(
      withCores({ ...ALPHA, clusterKey: "cluster-1" }),
      Core.set({ ...BASE, key: ALPHA.key, name: "Renamed" }),
    );
    expect(state.cores[ALPHA.key].clusterKey).toBe("cluster-1");
  });

  // Dropping the key here would orphan the cluster's stored state before a connection
  // could say whether the new address still reaches it.
  it.each([
    ["host", { host: "elsewhere.com" }],
    ["port", { port: 9099 }],
    ["scheme", { secure: true }],
  ])("should keep the cached cluster when the %s changes", (_, change) => {
    const state = reduce(
      withCores({ ...ALPHA, clusterKey: "cluster-1" }),
      Core.set({ ...BASE, ...change, key: ALPHA.key }),
    );
    expect(state.cores[ALPHA.key].clusterKey).toBe("cluster-1");
  });

  // Actions cross the window IPC boundary as plain JSON, so the payload's type is not
  // the only thing that can carry a field.
  it("should ignore a cluster the caller passes in", () => {
    const state = reduce(
      withCores(ALPHA),
      Core.set({ ...ALPHA, clusterKey: "cluster-1" } as Core.SetPayload),
    );
    expect(state.cores[ALPHA.key].clusterKey).toBeUndefined();
  });
});

describe("setClusterKey", () => {
  it("should cache the cluster a Core connected to", () => {
    const state = reduce(
      withCores(ALPHA),
      Core.setClusterKey({ key: ALPHA.key, clusterKey: "cluster-1" }),
    );
    expect(state.cores[ALPHA.key].clusterKey).toBe("cluster-1");
  });

  it("should replace the cluster when the address serves another one", () => {
    const state = reduce(
      withCores({ ...ALPHA, clusterKey: "cluster-1" }),
      Core.setClusterKey({ key: ALPHA.key, clusterKey: "cluster-2" }),
    );
    expect(state.cores[ALPHA.key].clusterKey).toBe("cluster-2");
  });

  it("should drop a cluster key for a missing Core", () => {
    const state = reduce(
      Core.ZERO_SLICE_STATE,
      Core.setClusterKey({ key: "nowhere", clusterKey: "cluster-1" }),
    );
    expect(state.cores.nowhere).toBeUndefined();
  });
});

describe("defaults", () => {
  it("should start with the local and demo Cores under their own keys", () => {
    expect(Object.keys(Core.ZERO_SLICE_STATE.cores).sort()).toEqual(
      [Core.DEMO_KEY, Core.LOCAL_KEY].sort(),
    );
  });
});

describe("remove", () => {
  it("should remove a single Core by key", () => {
    const state = reduce(withCores(ALPHA), Core.remove(ALPHA.key));
    expect(state.cores[ALPHA.key]).toBeUndefined();
  });

  it("should remove multiple Cores by key", () => {
    const state = reduce(withCores(ALPHA, BRAVO), Core.remove([ALPHA.key, BRAVO.key]));
    expect(Object.keys(state.cores)).toHaveLength(0);
  });

  it("should leave the defaults untouched when removing a missing key", () => {
    const state = reduce(Core.ZERO_SLICE_STATE, Core.remove("nowhere"));
    expect(Object.keys(state.cores)).toEqual(Object.keys(Core.ZERO_SLICE_STATE.cores));
  });

  it("should clear the selection when the selected Core is removed", () => {
    let state = reduce(withCores(ALPHA), Core.select(ALPHA.key));
    state = reduce(state, Core.remove(ALPHA.key));
    expect(state.selected).toBeUndefined();
  });

  it("should keep the selection when a different Core is removed", () => {
    let state = reduce(withCores(ALPHA, BRAVO), Core.select(ALPHA.key));
    state = reduce(state, Core.remove(BRAVO.key));
    expect(state.selected).toBe(ALPHA.key);
  });
});

describe("select / clearSelected", () => {
  it("should set the selected key", () => {
    const state = reduce(Core.ZERO_SLICE_STATE, Core.select(Core.DEMO_KEY));
    expect(state.selected).toBe(Core.DEMO_KEY);
  });

  it("should clear the selected key", () => {
    let state = reduce(Core.ZERO_SLICE_STATE, Core.select(Core.DEMO_KEY));
    state = reduce(state, Core.clearSelected());
    expect(state.selected).toBeUndefined();
  });
});

describe("coreZ", () => {
  // Releases through 0.57 stored whatever the connect form's text field produced, so
  // the same port sits on disk as both 9090 and "9090".
  it("should read a stored string port as a number", () => {
    const parsed = Core.coreZ.parse({
      key: "k",
      name: "Alpha",
      host: "localhost",
      port: "9090",
      username: "synnax",
      password: "seldon",
      secure: false,
    });
    expect(parsed.port).toBe(9090);
  });

  // The client's params carry behavioral fields; only the address may reach the disk.
  it("should store no field beyond the address and its cluster", () => {
    expect(Object.keys(Core.coreZ.shape).sort()).toEqual([
      "clusterKey",
      "host",
      "key",
      "name",
      "password",
      "port",
      "secure",
      "username",
    ]);
  });
});

describe("rename", () => {
  it("should rename a Core in place", () => {
    const state = reduce(
      withCores(ALPHA),
      Core.rename({ key: ALPHA.key, name: "Renamed" }),
    );
    expect(state.cores[ALPHA.key].name).toBe("Renamed");
  });

  // A name is a label, not an identity: the key is what everything resolves by.
  it("should allow two Cores to share a name", () => {
    const state = reduce(
      withCores(ALPHA, BRAVO),
      Core.rename({ key: ALPHA.key, name: BRAVO.name }),
    );
    expect(state.cores[ALPHA.key].name).toBe(BRAVO.name);
    expect(state.cores[BRAVO.key].name).toBe(BRAVO.name);
  });

  it("should drop a rename of a missing Core", () => {
    const state = reduce(
      Core.ZERO_SLICE_STATE,
      Core.rename({ key: "nowhere", name: "Alpha" }),
    );
    expect(state.cores.nowhere).toBeUndefined();
  });
});
