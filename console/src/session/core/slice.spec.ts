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
  order: cores.map((c) => c.key),
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

  it("should list a Core the user has just added last", () => {
    const state = reduce(withCores(BRAVO), Core.set({ ...BASE, name: "Alpha" }));
    expect(state.order).toHaveLength(2);
    expect(state.order[0]).toBe(BRAVO.key);
    expect(state.cores[state.order[1]].name).toBe("Alpha");
  });

  it("should keep the place of a Core that an edit replaces", () => {
    const state = reduce(
      withCores(ALPHA, BRAVO),
      Core.set({ ...BASE, key: ALPHA.key, name: "Zulu" }),
    );
    expect(state.order).toEqual([ALPHA.key, BRAVO.key]);
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

describe("setUser", () => {
  it("should label a Core with the user of its session", () => {
    const state = reduce(
      withCores(ALPHA),
      Core.setUser({ key: ALPHA.key, user: "Pat" }),
    );
    expect(state.cores[ALPHA.key].user).toBe("Pat");
  });

  it("should keep the label when an edit replaces the Core", () => {
    const state = reduce(
      withCores({ ...ALPHA, user: "Pat" }),
      Core.set({ ...BASE, key: ALPHA.key, name: "Renamed" }),
    );
    expect(state.cores[ALPHA.key].user).toBe("Pat");
  });

  it("should drop a label for a missing Core", () => {
    const state = reduce(
      Core.ZERO_SLICE_STATE,
      Core.setUser({ key: "nowhere", user: "Pat" }),
    );
    expect(state.cores.nowhere).toBeUndefined();
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
  it("should start with only the local Core", () => {
    expect(Object.keys(Core.ZERO_SLICE_STATE.cores)).toEqual(["LOCAL"]);
    expect(Core.ZERO_SLICE_STATE.order).toEqual(["LOCAL"]);
  });
});

describe("sliceStateZ", () => {
  const DEMO: Core.Core = {
    ...BASE,
    key: "DEMO",
    name: "Demo",
    host: "demo.synnaxlabs.com",
    secure: true,
  };

  it("should drop the retired demo Core from saved state", () => {
    const state = Core.sliceStateZ.parse({
      version: 0,
      selected: ALPHA.key,
      cores: { [ALPHA.key]: ALPHA, DEMO },
    });
    expect(state).toEqual({
      version: 0,
      selected: ALPHA.key,
      cores: { [ALPHA.key]: ALPHA },
      order: [ALPHA.key],
    });
  });

  it("should clear the selection when it named the retired demo Core", () => {
    const state = Core.sliceStateZ.parse({
      version: 0,
      selected: "DEMO",
      cores: { [ALPHA.key]: ALPHA, DEMO },
    });
    expect(state.selected).toBeUndefined();
  });

  it("should keep a demo record repointed at another host", () => {
    const repointed = { ...DEMO, host: "mine.example.com" };
    const state = Core.sliceStateZ.parse({
      version: 0,
      selected: "DEMO",
      cores: { DEMO: repointed },
    });
    expect(state.cores.DEMO).toEqual(repointed);
    expect(state.selected).toBe("DEMO");
  });

  it("should order saved Cores that have no order by name", () => {
    const state = Core.sliceStateZ.parse({
      version: 0,
      cores: { [BRAVO.key]: BRAVO, [ALPHA.key]: ALPHA },
    });
    expect(state.order).toEqual([ALPHA.key, BRAVO.key]);
  });

  it("should keep a saved order and add missing Cores at the end", () => {
    const CHARLIE: Core.Core = { ...BASE, key: "charlie", name: "Charlie" };
    const state = Core.sliceStateZ.parse({
      version: 0,
      cores: { [ALPHA.key]: ALPHA, [BRAVO.key]: BRAVO, [CHARLIE.key]: CHARLIE },
      order: [CHARLIE.key, "removed", BRAVO.key],
    });
    expect(state.order).toEqual([CHARLIE.key, BRAVO.key, ALPHA.key]);
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
    expect(state.order).toEqual([]);
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

describe("reorder", () => {
  const CHARLIE: Core.Core = { ...BASE, key: "charlie", name: "Charlie" };

  it("should move a Core down to the place of the one it drops on", () => {
    const state = reduce(
      withCores(ALPHA, BRAVO, CHARLIE),
      Core.reorder({ key: ALPHA.key, index: 2 }),
    );
    expect(state.order).toEqual([BRAVO.key, CHARLIE.key, ALPHA.key]);
  });

  it("should move a Core up to the place of the one it drops on", () => {
    const state = reduce(
      withCores(ALPHA, BRAVO, CHARLIE),
      Core.reorder({ key: CHARLIE.key, index: 0 }),
    );
    expect(state.order).toEqual([CHARLIE.key, ALPHA.key, BRAVO.key]);
  });

  it("should drop a move of a missing Core", () => {
    const state = reduce(withCores(ALPHA, BRAVO), Core.reorder({ key: "x", index: 0 }));
    expect(state.order).toEqual([ALPHA.key, BRAVO.key]);
  });
});

describe("select / clearSelected", () => {
  it("should set the selected key", () => {
    const state = reduce(withCores(ALPHA), Core.select(ALPHA.key));
    expect(state.selected).toBe(ALPHA.key);
  });

  it("should clear the selected key", () => {
    let state = reduce(withCores(ALPHA), Core.select(ALPHA.key));
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
      "lastLogin",
      "name",
      "oidc",
      "password",
      "port",
      "secure",
      "user",
      "username",
    ]);
  });
});

describe("PERSIST_EXCLUDE", () => {
  it("should keep every password out of the stored state", () => {
    const state = {
      [Core.SLICE_NAME]: withCores(
        { ...ALPHA, password: "secret" },
        { ...BRAVO, password: "other" },
      ),
    };
    const [purge] = Core.PERSIST_EXCLUDE;
    const purged = purge(state);
    const cores = Object.values(purged[Core.SLICE_NAME].cores);
    expect(cores.map((c) => c.password)).toEqual(["", ""]);
    expect(cores.map((c) => c.name)).toEqual([ALPHA.name, BRAVO.name]);
    expect(state[Core.SLICE_NAME].cores.alpha.password).toBe("secret");
  });

  it("should keep every OpenID Connect code out of the stored state", () => {
    const oidc = {
      provider: "idp",
      code: "code",
      verifier: "verifier",
      redirectUri: "http://127.0.0.1:4242/callback",
    };
    const state = { [Core.SLICE_NAME]: withCores({ ...ALPHA, oidc }) };
    const [purge] = Core.PERSIST_EXCLUDE;
    expect(purge(state)[Core.SLICE_NAME].cores.alpha).not.toHaveProperty("oidc");
    expect(state[Core.SLICE_NAME].cores.alpha.oidc).toEqual(oidc);
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

  it("should keep a renamed Core in its place", () => {
    const state = reduce(
      withCores(ALPHA, BRAVO),
      Core.rename({ key: ALPHA.key, name: "Zulu" }),
    );
    expect(state.order).toEqual([ALPHA.key, BRAVO.key]);
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

describe("recordLogin", () => {
  it("should stamp the login and select the Core, keeping later edits", () => {
    const state = reduce(
      withCores({ ...ALPHA, name: "Renamed", username: "ada", password: "pw" }),
      Core.recordLogin({ key: ALPHA.key, at: 42 }),
    );
    expect(state.cores[ALPHA.key]).toMatchObject({
      name: "Renamed",
      username: "",
      password: "",
      lastLogin: 42,
    });
    expect(state.cores[ALPHA.key].oidc).toBeUndefined();
    expect(state.selected).toBe(ALPHA.key);
  });

  it("should not restore a Core removed during the login", () => {
    const state = reduce(
      withCores(BRAVO),
      Core.recordLogin({ key: ALPHA.key, at: 42 }),
    );
    expect(state.cores[ALPHA.key]).toBeUndefined();
    expect(state.selected).toBeUndefined();
  });
});
