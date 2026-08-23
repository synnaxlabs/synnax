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

const reduce = (state: Core.SliceState, action: Core.Action): Core.SliceState =>
  Core.reducer(state, action);

const withCores = (...cores: Core.Core[]): Core.SliceState => ({
  ...Core.ZERO_SLICE_STATE,
  cores: Object.fromEntries(cores.map((c) => [c.key, c])),
});

describe("Core.key", () => {
  it("should key a Core by its host and port", () => {
    expect(Core.key({ host: "example.com", port: 9090 })).toBe("example.com:9090");
  });

  it("should give two Cores at the same address the same key", () => {
    expect(Core.keyed({ ...BASE, name: "A" }).key).toBe(
      Core.keyed({ ...BASE, name: "B" }).key,
    );
  });

  it("should separate Cores that differ only in port", () => {
    expect(Core.keyed(BASE).key).not.toBe(Core.keyed({ ...BASE, port: 9091 }).key);
  });
});

describe("set", () => {
  it("should add a Core under the key its address implies", () => {
    const state = reduce(Core.ZERO_SLICE_STATE, Core.set(BASE));
    expect(state.cores["example.com:9090"]).toMatchObject({ name: "My Core" });
  });

  it("should replace the entry at the same address rather than add a twin", () => {
    let state = reduce(Core.ZERO_SLICE_STATE, Core.set(BASE));
    state = reduce(state, Core.set({ ...BASE, name: "Renamed" }));
    const atAddress = Object.values(state.cores).filter(
      ({ host, port }) => host === BASE.host && port === BASE.port,
    );
    expect(atAddress).toHaveLength(1);
    expect(atAddress[0].name).toBe("Renamed");
  });

  it("should keep Cores that differ only in the secure flag apart from nothing", () => {
    let state = reduce(Core.ZERO_SLICE_STATE, Core.set(BASE));
    state = reduce(state, Core.set({ ...BASE, secure: true }));
    expect(state.cores["example.com:9090"].secure).toBe(true);
  });

  it("should move the entry when an edit changes the address", () => {
    let state = reduce(Core.ZERO_SLICE_STATE, Core.set(BASE));
    state = reduce(
      state,
      Core.set({ ...BASE, port: 9091, prevKey: "example.com:9090" }),
    );
    expect(state.cores["example.com:9090"]).toBeUndefined();
    expect(state.cores["example.com:9091"]).toBeDefined();
  });

  it("should follow the selection when an edit moves the selected Core", () => {
    let state = reduce(Core.ZERO_SLICE_STATE, Core.set(BASE));
    state = reduce(state, Core.select("example.com:9090"));
    state = reduce(
      state,
      Core.set({ ...BASE, port: 9091, prevKey: "example.com:9090" }),
    );
    expect(state.selected).toBe("example.com:9091");
  });
});

describe("remove", () => {
  it("should remove a single Core by key", () => {
    const alpha = Core.keyed({ ...BASE, name: "Alpha" });
    const state = reduce(withCores(alpha), Core.remove(alpha.key));
    expect(state.cores[alpha.key]).toBeUndefined();
  });

  it("should remove multiple Cores by key", () => {
    const alpha = Core.keyed({ ...BASE, name: "Alpha" });
    const beta = Core.keyed({ ...BASE, name: "Beta", port: 9091 });
    const state = reduce(withCores(alpha, beta), Core.remove([alpha.key, beta.key]));
    expect(Object.keys(state.cores)).toHaveLength(0);
  });

  it("should leave the defaults untouched when removing a missing key", () => {
    const state = reduce(Core.ZERO_SLICE_STATE, Core.remove("nowhere:1"));
    expect(Object.keys(state.cores)).toEqual(Object.keys(Core.ZERO_SLICE_STATE.cores));
  });

  it("should clear the selection when the selected Core is removed", () => {
    const alpha = Core.keyed({ ...BASE, name: "Alpha" });
    let state = reduce(withCores(alpha), Core.select(alpha.key));
    state = reduce(state, Core.remove(alpha.key));
    expect(state.selected).toBeUndefined();
  });

  it("should keep the selection when a different Core is removed", () => {
    const alpha = Core.keyed({ ...BASE, name: "Alpha" });
    const beta = Core.keyed({ ...BASE, name: "Beta", port: 9091 });
    let state = reduce(withCores(alpha, beta), Core.select(alpha.key));
    state = reduce(state, Core.remove(beta.key));
    expect(state.selected).toBe(alpha.key);
  });
});

describe("select / clearSelected", () => {
  it("should set the selected key", () => {
    const state = reduce(Core.ZERO_SLICE_STATE, Core.select("example.com:9090"));
    expect(state.selected).toBe("example.com:9090");
  });

  it("should clear the selected key", () => {
    let state = reduce(Core.ZERO_SLICE_STATE, Core.select("example.com:9090"));
    state = reduce(state, Core.clearSelected());
    expect(state.selected).toBeUndefined();
  });
});

describe("rename", () => {
  it("should rename a Core in place", () => {
    const alpha = Core.keyed({ ...BASE, name: "Alpha" });
    const state = reduce(
      withCores(alpha),
      Core.rename({ key: alpha.key, name: "Renamed" }),
    );
    expect(state.cores[alpha.key].name).toBe("Renamed");
  });

  it("should drop a rename to a name owned by another Core", () => {
    const alpha = Core.keyed({ ...BASE, name: "Alpha" });
    const beta = Core.keyed({ ...BASE, name: "Beta", port: 9091 });
    const state = reduce(
      withCores(alpha, beta),
      Core.rename({ key: alpha.key, name: "Beta" }),
    );
    expect(state.cores[alpha.key].name).toBe("Alpha");
  });

  it("should keep a rename to the Core's own name", () => {
    const alpha = Core.keyed({ ...BASE, name: "Alpha" });
    const state = reduce(
      withCores(alpha),
      Core.rename({ key: alpha.key, name: "Alpha" }),
    );
    expect(state.cores[alpha.key].name).toBe("Alpha");
  });

  it("should drop a rename of a missing Core", () => {
    const state = reduce(
      Core.ZERO_SLICE_STATE,
      Core.rename({ key: "nowhere", name: "Alpha" }),
    );
    expect(state.cores.nowhere).toBeUndefined();
  });
});
